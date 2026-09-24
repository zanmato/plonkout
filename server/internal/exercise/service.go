// Package exercise owns a user's exercise list.
package exercise

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
	"github.com/zanmato/plonkout/server/internal/exercise/exercisedb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
)

// Exercise is one entry of the exercise list.
type Exercise struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	MuscleGroup string    `json:"muscleGroup"`
	Type        string    `json:"type" enum:"strength,cardio"`
	DisplayType string    `json:"displayType" enum:"reps,time"`
	SingleArm   bool      `json:"singleArm"`
	Archived    bool      `json:"archived"`
	Created     time.Time `json:"created"`
}

// ExerciseInput is what a client writes.
type ExerciseInput struct {
	Name        string `json:"name" minLength:"1" maxLength:"100"`
	MuscleGroup string `json:"muscleGroup" maxLength:"50"`
	Type        string `json:"type" enum:"strength,cardio"`
	DisplayType string `json:"displayType" enum:"reps,time"`
	SingleArm   bool   `json:"singleArm"`
	Archived    bool   `json:"archived,omitempty"`
}

// Service is the exercise module.
type Service struct {
	pool *pgxpool.Pool
	q    *exercisedb.Queries
}

// NewService builds the exercise module.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: exercisedb.New(pool)}
}

// List returns every exercise, archived ones included.
func (s *Service) List(ctx context.Context) ([]Exercise, error) {
	rows, err := s.q.ListExercises(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Exercise, len(rows))
	for i, row := range rows {
		out[i] = toExercise(row)
	}
	return out, nil
}

// Create adds an exercise. Names are unique per user, ignoring case.
func (s *Service) Create(ctx context.Context, in ExerciseInput) (Exercise, error) {
	row, err := s.q.CreateExercise(ctx, exercisedb.CreateExerciseParams{
		Name: strings.TrimSpace(in.Name), MuscleGroup: in.MuscleGroup, Type: in.Type,
		DisplayType: in.DisplayType, SingleArm: in.SingleArm,
	})
	if err != nil {
		return Exercise{}, duplicateName(err)
	}
	return toExercise(row), nil
}

// Update changes an exercise. A rename carries the logged history and the
// block periodization state along, since both are keyed by name.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in ExerciseInput) (Exercise, error) {
	var out Exercise
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		before, err := q.GetExercise(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no such exercise", api.ErrNotFound)
		} else if err != nil {
			return err
		}
		name := strings.TrimSpace(in.Name)
		row, err := q.UpdateExercise(ctx, exercisedb.UpdateExerciseParams{
			ID: id, Name: name, MuscleGroup: in.MuscleGroup, Type: in.Type,
			DisplayType: in.DisplayType, SingleArm: in.SingleArm, Archived: in.Archived,
		})
		if err != nil {
			return duplicateName(err)
		}
		if before.Name != name {
			if err := q.RenameExerciseSnapshots(ctx, exercisedb.RenameExerciseSnapshotsParams{
				ExerciseID: &id, OldName: before.Name, NewName: name,
			}); err != nil {
				return err
			}
			if err := q.RenameBlockPeriodizationKey(ctx, exercisedb.RenameBlockPeriodizationKeyParams{
				OldName: before.Name, NewName: name,
			}); err != nil {
				return err
			}
		}
		out = toExercise(row)
		return nil
	})
	return out, err
}

// Resolve finds an exercise by name, ignoring case. An unknown name answers
// with the closest names, so a typo is corrected rather than stored.
func (s *Service) Resolve(ctx context.Context, name string) (Exercise, error) {
	row, err := s.q.FindExerciseByName(ctx, strings.TrimSpace(name))
	if err == nil {
		return toExercise(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Exercise{}, err
	}
	suggestions, err := s.q.SuggestExercises(ctx, name)
	if err != nil {
		return Exercise{}, err
	}
	detail := fmt.Sprintf("no exercise is called %q", name)
	if len(suggestions) > 0 {
		detail += ", did you mean " + strings.Join(quoteAll(suggestions), " or ") + "?"
	}
	return Exercise{}, api.Errorf(http.StatusUnprocessableEntity, "unknown_exercise", "%s", detail)
}

func quoteAll(names []string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = fmt.Sprintf("%q", name)
	}
	return out
}

func duplicateName(err error) error {
	var problem = api.MapError(err)
	if problem.Code == "already_exists" {
		return api.Errorf(http.StatusConflict, "exercise_exists", "an exercise with that name already exists")
	}
	return err
}

func toExercise(row exercisedb.Exercise) Exercise {
	return Exercise{
		ID: row.ID, Name: row.Name, MuscleGroup: row.MuscleGroup, Type: row.Type,
		DisplayType: row.DisplayType, SingleArm: row.SingleArm, Archived: row.Archived, Created: row.CreatedAt,
	}
}
