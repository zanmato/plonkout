// Package workout stores logged workouts.
//
// A workout is read and written as one document, the way the editor autosaves
// it. The rows underneath are normalized so history, statistics and a plan's
// planned versus actual comparison are plain SQL.
package workout

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/workout/workoutdb"
)

// Service is the workout module.
type Service struct {
	pool *pgxpool.Pool
	q    *workoutdb.Queries
}

// NewService builds the workout module.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: workoutdb.New(pool)}
}

// Filter narrows a list of workouts. Every field is optional.
type Filter struct {
	From, To *time.Time
	// Exercise keeps workouts that logged an exercise of this name.
	Exercise *string
	Limit    *int32
}

// List returns workouts newest first.
func (s *Service) List(ctx context.Context, f Filter) ([]Workout, error) {
	rows, err := s.q.ListWorkouts(ctx, workoutdb.ListWorkoutsParams{
		FromTime: f.From, ToTime: f.To, Exercise: f.Exercise, MaxRows: f.Limit,
	})
	if err != nil {
		return nil, err
	}
	return load(ctx, s.q, rows)
}

// Get returns one workout.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Workout, error) {
	row, err := s.q.GetWorkout(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workout{}, fmt.Errorf("%w: no such workout", api.ErrNotFound)
	} else if err != nil {
		return Workout{}, err
	}
	return loadOne(ctx, s.q, row)
}

// Latest returns the most recent workout with this name, skipping excludeID.
func (s *Service) Latest(ctx context.Context, name string, excludeID *uuid.UUID) (Workout, error) {
	row, err := s.q.LatestWorkoutByName(ctx, workoutdb.LatestWorkoutByNameParams{Name: name, ExcludeID: excludeID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Workout{}, fmt.Errorf("%w: no workout with that name", api.ErrNotFound)
	} else if err != nil {
		return Workout{}, err
	}
	return loadOne(ctx, s.q, row)
}

// Create stores a new workout.
func (s *Service) Create(ctx context.Context, in WorkoutInput) (Workout, error) {
	var out Workout
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		created, err := CreateInTx(ctx, tx, in, CreateOptions{})
		out = created
		return err
	})
	return out, err
}

// CreateOptions are what an import or a planned session adds to a new workout.
type CreateOptions struct {
	PlannedSessionID *uuid.UUID
	LegacyID         *int64
	Created          *time.Time
	Updated          *time.Time
}

// ErrAlreadyImported is returned by CreateInTx for a legacy id that exists.
var ErrAlreadyImported = errors.New("already imported")

// CreateInTx stores a new workout inside the caller's transaction.
func CreateInTx(ctx context.Context, tx pgx.Tx, in WorkoutInput, opts CreateOptions) (Workout, error) {
	q := workoutdb.New(tx)
	id, err := uuid.NewV7()
	if err != nil {
		return Workout{}, err
	}
	row, err := q.CreateWorkout(ctx, workoutdb.CreateWorkoutParams{
		ID:               id,
		Name:             in.Name,
		Started:          in.Started,
		Ended:            in.Ended,
		Notes:            in.Notes,
		PlannedSessionID: opts.PlannedSessionID,
		LegacyID:         opts.LegacyID,
		Created:          opts.Created,
		Updated:          opts.Updated,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT on the legacy id did nothing.
		return Workout{}, ErrAlreadyImported
	} else if err != nil {
		return Workout{}, err
	}
	exercises, err := writeExercises(ctx, q, row.ID, in.Exercises)
	if err != nil {
		return Workout{}, err
	}
	if err := syncSession(ctx, q, row); err != nil {
		return Workout{}, err
	}
	return toWorkout(row, exercises), nil
}

// Update replaces a workout, as long as the client saw the current revision.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in WorkoutUpdate) (Workout, error) {
	var out Workout
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := q.UpdateWorkout(ctx, workoutdb.UpdateWorkoutParams{
			ID: id, Revision: in.Revision, Name: in.Name, Started: in.Started, Ended: in.Ended, Notes: in.Notes,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			if _, getErr := q.GetWorkout(ctx, id); errors.Is(getErr, pgx.ErrNoRows) {
				return fmt.Errorf("%w: no such workout", api.ErrNotFound)
			}
			return api.Errorf(http.StatusConflict, "stale_revision",
				"the workout was changed somewhere else, reload it before saving")
		} else if err != nil {
			return err
		}
		if err := q.DeleteWorkoutExercises(ctx, id); err != nil {
			return err
		}
		exercises, err := writeExercises(ctx, q, id, in.Exercises)
		if err != nil {
			return err
		}
		if err := syncSession(ctx, q, row); err != nil {
			return err
		}
		out = toWorkout(row, exercises)
		return nil
	})
	return out, err
}

// Delete removes a workout and its sets. A planned session it was logged for
// goes back in the queue.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		sessionID, err := q.DeleteWorkout(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no such workout", api.ErrNotFound)
		} else if err != nil {
			return err
		}
		if sessionID != nil {
			return q.ReleaseSession(ctx, *sessionID)
		}
		return nil
	})
}

// syncSession keeps the planned session of a workout in step with it: in
// progress until the workout has ended, completed after.
func syncSession(ctx context.Context, q *workoutdb.Queries, row workoutdb.Workout) error {
	if row.PlannedSessionID == nil {
		return nil
	}
	return q.SyncSessionStatus(ctx, workoutdb.SyncSessionStatusParams{ID: *row.PlannedSessionID, Ended: row.Ended})
}

// writeExercises inserts the exercises and sets of a workout in two round
// trips, and answers with what was stored, linked to the exercise list.
func writeExercises(ctx context.Context, q *workoutdb.Queries, workoutID uuid.UUID, exercises []WorkoutExercise) ([]WorkoutExercise, error) {
	if len(exercises) == 0 {
		return []WorkoutExercise{}, nil
	}
	if err := linkExercises(ctx, q, exercises); err != nil {
		return nil, err
	}

	exerciseParams := make([]workoutdb.InsertWorkoutExerciseParams, len(exercises))
	var setParams []workoutdb.InsertWorkoutSetParams
	for i, e := range exercises {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		exerciseParams[i] = workoutdb.InsertWorkoutExerciseParams{
			ID:                id,
			WorkoutID:         workoutID,
			Position:          int32(i),
			ExerciseID:        e.ExerciseID,
			Name:              strings.TrimSpace(e.Name),
			MuscleGroup:       e.MuscleGroup,
			Type:              e.Type,
			DisplayType:       e.DisplayType,
			SingleArm:         e.SingleArm,
			Intensity:         e.Intensity,
			PlannedExerciseID: e.PlannedExerciseID,
			Notes:             e.Notes,
		}
		for j, set := range e.Sets {
			setParams = append(setParams, workoutdb.InsertWorkoutSetParams{
				WorkoutExerciseID: id,
				Position:          int32(j),
				Type:              set.Type,
				Weight:            set.Weight,
				Distance:          set.Distance,
				Reps:              set.Reps,
				Time:              set.Time,
				Rpe:               set.RPE,
				Arm:               set.Arm,
				Notes:             set.Notes,
				TargetID:          set.TargetID,
				TargetSeq:         set.TargetSeq,
			})
		}
	}

	if err := batchErr(q.InsertWorkoutExercise(ctx, exerciseParams)); err != nil {
		return nil, fmt.Errorf("store workout exercises: %w", err)
	}
	if len(setParams) > 0 {
		if err := batchErr(q.InsertWorkoutSet(ctx, setParams)); err != nil {
			return nil, fmt.Errorf("store workout sets: %w", err)
		}
	}
	return exercises, nil
}

type batch interface {
	Exec(func(int, error))
}

func batchErr(b batch) error {
	var first error
	b.Exec(func(_ int, err error) {
		if err != nil && first == nil {
			first = err
		}
	})
	return first
}

// linkExercises fills in exercise ids the client left out, by name.
func linkExercises(ctx context.Context, q *workoutdb.Queries, exercises []WorkoutExercise) error {
	var names []string
	for _, e := range exercises {
		if e.ExerciseID == nil {
			names = append(names, strings.ToLower(strings.TrimSpace(e.Name)))
		}
	}
	if len(names) == 0 {
		return nil
	}
	rows, err := q.ExerciseIDsByName(ctx, names)
	if err != nil {
		return err
	}
	byName := make(map[string]uuid.UUID, len(rows))
	for _, row := range rows {
		byName[row.Key] = row.ID
	}
	for i := range exercises {
		if exercises[i].ExerciseID != nil {
			continue
		}
		if id, ok := byName[strings.ToLower(strings.TrimSpace(exercises[i].Name))]; ok {
			exercises[i].ExerciseID = &id
		}
	}
	return nil
}

func loadOne(ctx context.Context, q *workoutdb.Queries, row workoutdb.Workout) (Workout, error) {
	workouts, err := load(ctx, q, []workoutdb.Workout{row})
	if err != nil {
		return Workout{}, err
	}
	return workouts[0], nil
}

// load assembles workouts from three queries, however many workouts there are.
func load(ctx context.Context, q *workoutdb.Queries, rows []workoutdb.Workout) ([]Workout, error) {
	out := make([]Workout, len(rows))
	if len(rows) == 0 {
		return out, nil
	}

	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	exerciseRows, err := q.ListWorkoutExercises(ctx, ids)
	if err != nil {
		return nil, err
	}
	exerciseIDs := make([]uuid.UUID, len(exerciseRows))
	for i, row := range exerciseRows {
		exerciseIDs[i] = row.ID
	}
	setRows, err := q.ListWorkoutSets(ctx, exerciseIDs)
	if err != nil {
		return nil, err
	}

	sets := make(map[uuid.UUID][]WorkoutSet, len(exerciseRows))
	for _, row := range setRows {
		sets[row.WorkoutExerciseID] = append(sets[row.WorkoutExerciseID], WorkoutSet{
			Type:      row.Type,
			Weight:    row.Weight,
			Distance:  row.Distance,
			Reps:      row.Reps,
			Time:      row.Time,
			RPE:       row.Rpe,
			Arm:       row.Arm,
			Notes:     row.Notes,
			TargetID:  row.TargetID,
			TargetSeq: row.TargetSeq,
		})
	}
	exercises := make(map[uuid.UUID][]WorkoutExercise, len(rows))
	for _, row := range exerciseRows {
		workoutSets := sets[row.ID]
		if workoutSets == nil {
			workoutSets = []WorkoutSet{}
		}
		exercises[row.WorkoutID] = append(exercises[row.WorkoutID], WorkoutExercise{
			ExerciseID:        row.ExerciseID,
			Name:              row.Name,
			MuscleGroup:       row.MuscleGroup,
			Type:              row.Type,
			DisplayType:       row.DisplayType,
			SingleArm:         row.SingleArm,
			Intensity:         row.Intensity,
			Notes:             row.Notes,
			PlannedExerciseID: row.PlannedExerciseID,
			Sets:              workoutSets,
		})
	}

	for i, row := range rows {
		out[i] = toWorkout(row, exercises[row.ID])
	}
	return out, nil
}

func toWorkout(row workoutdb.Workout, exercises []WorkoutExercise) Workout {
	if exercises == nil {
		exercises = []WorkoutExercise{}
	}
	return Workout{
		ID:               row.ID,
		Name:             row.Name,
		Started:          row.Started,
		Ended:            row.Ended,
		Notes:            row.Notes,
		PlannedSessionID: row.PlannedSessionID,
		Revision:         row.Revision,
		Created:          row.CreatedAt,
		Updated:          row.UpdatedAt,
		Exercises:        exercises,
	}
}
