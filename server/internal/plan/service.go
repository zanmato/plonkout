// Package plan owns training plans: an ordered queue of sessions with the
// exercises and targets each one prescribes. The user picks the next session
// in the app, which starts a workout linked back to it, so what was planned
// and what was done can be compared later.
package plan

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/exercise/exercisedb"
	"github.com/zanmato/plonkout/server/internal/plan/plandb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
)

// Session statuses.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusSkipped    = "skipped"
)

const dateLayout = "2006-01-02"

// Service is the plan module.
type Service struct {
	pool *pgxpool.Pool
	q    *plandb.Queries
}

// NewService builds the plan module.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: plandb.New(pool)}
}

// List returns plans, newest first, optionally of one status, with their
// sessions.
func (s *Service) List(ctx context.Context, status *string) ([]Plan, error) {
	rows, err := s.q.ListPlans(ctx, status)
	if err != nil {
		return nil, err
	}
	return s.load(ctx, s.q, rows, false)
}

// Get returns one plan. With actuals, every session that has a workout
// carries its planned versus actual comparison.
func (s *Service) Get(ctx context.Context, id uuid.UUID, actuals bool) (Plan, error) {
	row, err := s.q.GetPlan(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, fmt.Errorf("%w: no such plan", api.ErrNotFound)
	} else if err != nil {
		return Plan{}, err
	}
	plans, err := s.load(ctx, s.q, []plandb.Plan{row}, actuals)
	if err != nil {
		return Plan{}, err
	}
	return plans[0], nil
}

// Create stores a plan and its sessions.
func (s *Service) Create(ctx context.Context, in PlanInput) (Plan, error) {
	startDate, err := parseDate(in.StartDate)
	if err != nil {
		return Plan{}, err
	}
	var id uuid.UUID
	err = db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := q.CreatePlan(ctx, plandb.CreatePlanParams{
			Name: in.Name, Goal: in.Goal, Notes: in.Notes, StartDate: startDate,
		})
		if err != nil {
			return err
		}
		id = row.ID
		return insertSessions(ctx, tx, row.ID, -1, in.Sessions, in.CreateMissingExercises)
	})
	if err != nil {
		return Plan{}, err
	}
	return s.Get(ctx, id, false)
}

// Update changes a plan's own fields.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in PlanUpdate) (Plan, error) {
	startDate, err := parseDate(in.StartDate)
	if err != nil {
		return Plan{}, err
	}
	_, err = s.q.UpdatePlan(ctx, plandb.UpdatePlanParams{
		ID: id, Name: in.Name, Goal: in.Goal, Notes: in.Notes, Status: in.Status, StartDate: startDate,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, fmt.Errorf("%w: no such plan", api.ErrNotFound)
	} else if err != nil {
		return Plan{}, err
	}
	return s.Get(ctx, id, false)
}

// Delete removes a plan and its sessions. Workouts logged for it stay, only
// losing their link to the plan.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.q.DeletePlan(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("%w: no such plan", api.ErrNotFound)
	}
	return nil
}

// AddSessions inserts sessions after one, or at the end of the queue.
func (s *Service) AddSessions(ctx context.Context, planID uuid.UUID, after *uuid.UUID, sessions []SessionInput, createMissing bool) (Plan, error) {
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if _, err := q.GetPlan(ctx, planID); errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no such plan", api.ErrNotFound)
		} else if err != nil {
			return err
		}

		position := int32(-1)
		if after != nil {
			anchor, err := q.GetSessionForUpdate(ctx, *after)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && anchor.PlanID != planID) {
				return fmt.Errorf("%w: no such session in this plan", api.ErrNotFound)
			} else if err != nil {
				return err
			}
			position = anchor.Position
			if err := q.ShiftSessions(ctx, plandb.ShiftSessionsParams{
				PlanID: planID, After: position, By: int32(len(sessions)),
			}); err != nil {
				return err
			}
		} else {
			last, err := q.MaxSessionPosition(ctx, planID)
			if err != nil {
				return err
			}
			position = last
		}
		return insertSessions(ctx, tx, planID, position, sessions, createMissing)
	})
	if err != nil {
		return Plan{}, err
	}
	return s.Get(ctx, planID, false)
}

// UpdateSession changes a pending session. Exercises, when given, replace
// every planned exercise and target.
func (s *Service) UpdateSession(ctx context.Context, id uuid.UUID, in SessionUpdate) (Session, error) {
	var planID uuid.UUID
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		session, err := lockPending(ctx, q, id, "edited")
		if err != nil {
			return err
		}
		planID = session.PlanID
		label, week, day, intensity, notes := session.Label, session.Week, session.Day, session.Intensity, session.Notes
		if in.Label != nil {
			label = *in.Label
		}
		if in.Week != nil {
			week = in.Week
		}
		if in.Day != nil {
			day = in.Day
		}
		if in.Intensity != nil {
			intensity = in.Intensity
		}
		if in.Notes != nil {
			notes = *in.Notes
		}
		if err := q.UpdateSession(ctx, plandb.UpdateSessionParams{
			ID: id, Label: label, Week: week, Day: day, Intensity: intensity, Notes: notes,
		}); err != nil {
			return err
		}
		if in.Exercises == nil {
			return nil
		}
		if err := q.DeleteSessionExercises(ctx, id); err != nil {
			return err
		}
		resolver := newResolver(tx, in.CreateMissingExercises)
		return insertExercises(ctx, tx, resolver, id, in.Exercises)
	})
	if err != nil {
		return Session{}, err
	}
	return s.session(ctx, planID, id)
}

// SessionUpdate changes a pending session. Absent fields are kept.
type SessionUpdate struct {
	Label                  *string                `json:"label,omitempty" maxLength:"60"`
	Week                   *int32                 `json:"week,omitempty" minimum:"1"`
	Day                    *string                `json:"day,omitempty" maxLength:"8"`
	Intensity              *string                `json:"intensity,omitempty" enum:"heavy,light"`
	Notes                  *string                `json:"notes,omitempty" maxLength:"2000"`
	Exercises              []PlannedExerciseInput `json:"exercises,omitempty" maxItems:"30" doc:"Replaces every planned exercise and target of the session."`
	CreateMissingExercises bool                   `json:"createMissingExercises,omitempty"`
}

// Reorder sets the order of a plan's pending sessions. The ids must be
// exactly the pending sessions. Sessions already started, done or skipped
// keep their place.
func (s *Service) Reorder(ctx context.Context, planID uuid.UUID, ids []uuid.UUID) (Plan, error) {
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		sessions, err := q.ListSessions(ctx, []uuid.UUID{planID})
		if err != nil {
			return err
		}
		var positions []int32
		pending := map[uuid.UUID]bool{}
		for _, session := range sessions {
			if session.Status == StatusPending {
				positions = append(positions, session.Position)
				pending[session.ID] = true
			}
		}
		if len(ids) != len(pending) {
			return api.Errorf(http.StatusUnprocessableEntity, "invalid_order",
				"list every pending session of the plan exactly once, there are %d", len(pending))
		}
		seen := map[uuid.UUID]bool{}
		for _, id := range ids {
			if !pending[id] || seen[id] {
				return api.Errorf(http.StatusUnprocessableEntity, "invalid_order",
					"session %s is not a pending session of this plan, or is listed twice", id)
			}
			seen[id] = true
		}
		// The pending sessions trade the positions they already hold, so the
		// others never move. The unique constraint is deferred to commit.
		for i, id := range ids {
			if err := q.SetSessionPosition(ctx, plandb.SetSessionPositionParams{ID: id, Position: positions[i]}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Plan{}, err
	}
	return s.Get(ctx, planID, false)
}

// SetStatus skips a pending session, puts a skipped one back, or deletes a
// session that was never started.
func (s *Service) SetStatus(ctx context.Context, id uuid.UUID, status, reason string) (*Session, error) {
	var planID uuid.UUID
	deleted := false
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		session, err := q.GetSessionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no such session", api.ErrNotFound)
		} else if err != nil {
			return err
		}
		planID = session.PlanID

		switch status {
		case StatusSkipped:
			if session.Status != StatusPending {
				return notPending(session.Status, "skipped")
			}
			return q.SetSessionStatus(ctx, plandb.SetSessionStatusParams{ID: id, Status: StatusSkipped, SkipReason: reason})
		case StatusPending:
			if session.Status != StatusSkipped && session.Status != StatusPending {
				return api.Errorf(http.StatusConflict, "session_started",
					"the session is %s, only a skipped session can be put back", session.Status)
			}
			return q.SetSessionStatus(ctx, plandb.SetSessionStatusParams{ID: id, Status: StatusPending})
		case "deleted":
			if session.Status != StatusPending && session.Status != StatusSkipped {
				return api.Errorf(http.StatusConflict, "session_started",
					"the session is %s and has a workout, it cannot be deleted", session.Status)
			}
			deleted = true
			_, err := q.DeleteSession(ctx, id)
			return err
		default:
			return api.Errorf(http.StatusUnprocessableEntity, "invalid_status", "unknown status %q", status)
		}
	})
	if err != nil || deleted {
		return nil, err
	}
	session, err := s.session(ctx, planID, id)
	return &session, err
}

// Queue returns the sessions still to do in active plans, in order.
func (s *Service) Queue(ctx context.Context) ([]QueueEntry, error) {
	rows, err := s.q.ListQueue(ctx)
	if err != nil {
		return nil, err
	}
	sessions := make([]plandb.ListSessionsRow, len(rows))
	for i, row := range rows {
		sessions[i] = plandb.ListSessionsRow{
			ID: row.ID, UserID: row.UserID, PlanID: row.PlanID, Position: row.Position, Label: row.Label,
			Week: row.Week, Day: row.Day, Intensity: row.Intensity, Notes: row.Notes, Status: row.Status,
			SkipReason: row.SkipReason, CompletedAt: row.CompletedAt, CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt, WorkoutID: row.WorkoutID,
		}
	}
	loaded, err := loadSessions(ctx, s.q, sessions, false)
	if err != nil {
		return nil, err
	}
	out := make([]QueueEntry, len(rows))
	for i, row := range rows {
		out[i] = QueueEntry{PlanName: row.PlanName, Session: loaded[i]}
	}
	return out, nil
}

// QueueEntry is a session to do, with the plan it belongs to.
type QueueEntry struct {
	PlanName string  `json:"planName"`
	Session  Session `json:"session"`
}

// Comparison returns planned versus actual for one session.
func (s *Service) Comparison(ctx context.Context, id uuid.UUID) (Comparison, error) {
	row, err := s.q.GetSession(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Comparison{}, fmt.Errorf("%w: no such session", api.ErrNotFound)
	} else if err != nil {
		return Comparison{}, err
	}
	sessions, err := loadSessions(ctx, s.q, []plandb.ListSessionsRow{plandb.ListSessionsRow(row)}, true)
	if err != nil {
		return Comparison{}, err
	}
	if sessions[0].Comparison == nil {
		return Comparison{Exercises: []ExerciseComparison{}, Unplanned: []string{}}, nil
	}
	return *sessions[0].Comparison, nil
}

func (s *Service) session(ctx context.Context, planID, id uuid.UUID) (Session, error) {
	p, err := s.Get(ctx, planID, false)
	if err != nil {
		return Session{}, err
	}
	for _, session := range p.Sessions {
		if session.ID == id {
			return session, nil
		}
	}
	return Session{}, fmt.Errorf("%w: no such session", api.ErrNotFound)
}

func lockPending(ctx context.Context, q *plandb.Queries, id uuid.UUID, action string) (plandb.PlannedSession, error) {
	session, err := q.GetSessionForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return session, fmt.Errorf("%w: no such session", api.ErrNotFound)
	} else if err != nil {
		return session, err
	}
	if session.Status != StatusPending {
		return session, notPending(session.Status, action)
	}
	return session, nil
}

func notPending(status, action string) error {
	return api.Errorf(http.StatusConflict, "session_not_pending",
		"the session is %s, only a pending session can be %s", status, action)
}

func insertSessions(ctx context.Context, tx pgx.Tx, planID uuid.UUID, after int32, sessions []SessionInput, createMissing bool) error {
	q := plandb.New(tx)
	resolver := newResolver(tx, createMissing)
	for i, in := range sessions {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		if err := q.InsertSession(ctx, plandb.InsertSessionParams{
			ID: id, PlanID: planID, Position: after + 1 + int32(i),
			Label: in.Label, Week: in.Week, Day: in.Day, Intensity: in.Intensity, Notes: in.Notes,
		}); err != nil {
			return err
		}
		if err := insertExercises(ctx, tx, resolver, id, in.Exercises); err != nil {
			return fmt.Errorf("session %d (%s): %w", i+1, in.Label, err)
		}
	}
	return nil
}

func insertExercises(ctx context.Context, tx pgx.Tx, resolver *resolver, sessionID uuid.UUID, exercises []PlannedExerciseInput) error {
	q := plandb.New(tx)
	var exerciseParams []plandb.InsertPlannedExerciseParams
	var targetParams []plandb.InsertTargetParams
	for i, in := range exercises {
		exerciseID, err := resolver.resolve(ctx, in)
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		exerciseParams = append(exerciseParams, plandb.InsertPlannedExerciseParams{
			ID: id, SessionID: sessionID, Position: int32(i), ExerciseID: exerciseID,
			Notes: in.Notes, OffArmPercent: in.OffArmPercent,
		})
		for j, t := range in.Targets {
			if t.RPEMin != nil && t.RPEMax != nil && *t.RPEMin > *t.RPEMax {
				return api.Errorf(http.StatusUnprocessableEntity, "invalid_target",
					"%s: rpeMin %.1f is above rpeMax %.1f", in.Exercise, *t.RPEMin, *t.RPEMax)
			}
			targetID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			targetParams = append(targetParams, plandb.InsertTargetParams{
				ID: targetID, PlannedExerciseID: id, Position: int32(j), SetType: t.SetType, Sets: t.Sets,
				Reps: t.Reps, Weight: t.Weight, Time: t.Time, RpeMin: t.RPEMin, RpeMax: t.RPEMax, Notes: t.Notes,
			})
		}
	}
	if len(exerciseParams) == 0 {
		return nil
	}
	if err := batchErr(q.InsertPlannedExercise(ctx, exerciseParams)); err != nil {
		return err
	}
	return batchErr(q.InsertTarget(ctx, targetParams))
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

// resolver turns exercise names into exercise ids, creating missing ones when
// allowed and suggesting close names when not.
type resolver struct {
	q             *exercisedb.Queries
	createMissing bool
	known         map[string]uuid.UUID
}

func newResolver(tx pgx.Tx, createMissing bool) *resolver {
	return &resolver{q: exercisedb.New(tx), createMissing: createMissing, known: map[string]uuid.UUID{}}
}

func (r *resolver) resolve(ctx context.Context, in PlannedExerciseInput) (uuid.UUID, error) {
	name := strings.TrimSpace(in.Exercise)
	key := strings.ToLower(name)
	if id, ok := r.known[key]; ok {
		return id, nil
	}
	row, err := r.q.FindExerciseByName(ctx, name)
	if err == nil {
		r.known[key] = row.ID
		return row.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	if !r.createMissing {
		suggestions, err := r.q.SuggestExercises(ctx, name)
		if err != nil {
			return uuid.Nil, err
		}
		detail := fmt.Sprintf("no exercise is called %q", name)
		if len(suggestions) > 0 {
			quoted := make([]string, len(suggestions))
			for i, s := range suggestions {
				quoted[i] = fmt.Sprintf("%q", s)
			}
			detail += ", did you mean " + strings.Join(quoted, " or ") + "? Set createMissingExercises to add it instead"
		}
		return uuid.Nil, api.Errorf(http.StatusUnprocessableEntity, "unknown_exercise", "%s", detail)
	}
	singleArm := in.SingleArm != nil && *in.SingleArm
	created, err := r.q.CreateExercise(ctx, exercisedb.CreateExerciseParams{
		Name: name, MuscleGroup: in.MuscleGroup, Type: "strength", DisplayType: "reps", SingleArm: singleArm,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create exercise %q: %w", name, err)
	}
	r.known[key] = created.ID
	return created.ID, nil
}

// load assembles plans with their sessions.
func (s *Service) load(ctx context.Context, q *plandb.Queries, rows []plandb.Plan, actuals bool) ([]Plan, error) {
	out := make([]Plan, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	sessionRows, err := q.ListSessions(ctx, ids)
	if err != nil {
		return nil, err
	}
	sessions, err := loadSessions(ctx, q, sessionRows, actuals)
	if err != nil {
		return nil, err
	}
	byPlan := map[uuid.UUID][]Session{}
	for _, session := range sessions {
		byPlan[session.PlanID] = append(byPlan[session.PlanID], session)
	}
	for i, row := range rows {
		planSessions := byPlan[row.ID]
		if planSessions == nil {
			planSessions = []Session{}
		}
		out[i] = Plan{
			ID: row.ID, Name: row.Name, Goal: row.Goal, Notes: row.Notes, Status: row.Status,
			StartDate: formatDate(row.StartDate), Created: row.CreatedAt, Updated: row.UpdatedAt,
			Sessions: planSessions,
		}
	}
	return out, nil
}

func loadSessions(ctx context.Context, q *plandb.Queries, rows []plandb.ListSessionsRow, actuals bool) ([]Session, error) {
	out := make([]Session, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}

	exerciseRows, err := q.ListPlannedExercises(ctx, ids)
	if err != nil {
		return nil, err
	}
	exerciseIDs := make([]uuid.UUID, len(exerciseRows))
	for i, row := range exerciseRows {
		exerciseIDs[i] = row.ID
	}
	targetRows, err := q.ListTargets(ctx, exerciseIDs)
	if err != nil {
		return nil, err
	}

	targets := map[uuid.UUID][]Target{}
	for _, t := range targetRows {
		targets[t.PlannedExerciseID] = append(targets[t.PlannedExerciseID], Target{
			ID: t.ID, SetType: t.SetType, Sets: t.Sets, Reps: t.Reps, Weight: t.Weight, Time: t.Time,
			RPEMin: t.RpeMin, RPEMax: t.RpeMax, Notes: t.Notes,
		})
	}
	exercises := map[uuid.UUID][]PlannedExercise{}
	for _, e := range exerciseRows {
		exerciseTargets := targets[e.ID]
		if exerciseTargets == nil {
			exerciseTargets = []Target{}
		}
		exercises[e.SessionID] = append(exercises[e.SessionID], PlannedExercise{
			ID: e.ID, ExerciseID: e.ExerciseID, Exercise: e.ExerciseName, MuscleGroup: e.MuscleGroup,
			Type: e.ExerciseType, DisplayType: e.DisplayType, SingleArm: e.SingleArm,
			Notes: e.Notes, OffArmPercent: e.OffArmPercent, Targets: exerciseTargets,
		})
	}

	for i, row := range rows {
		sessionExercises := exercises[row.ID]
		if sessionExercises == nil {
			sessionExercises = []PlannedExercise{}
		}
		out[i] = Session{
			ID: row.ID, PlanID: row.PlanID, Position: row.Position, Label: row.Label, Week: row.Week,
			Day: row.Day, Intensity: row.Intensity, Notes: row.Notes, Status: row.Status,
			SkipReason: row.SkipReason, CompletedAt: row.CompletedAt, WorkoutID: row.WorkoutID,
			Exercises: sessionExercises,
		}
	}

	if actuals {
		if err := attachComparisons(ctx, q, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, *value)
	if err != nil {
		return nil, api.Errorf(http.StatusUnprocessableEntity, "invalid_date", "startDate must be YYYY-MM-DD")
	}
	return &t, nil
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

// sessionIDs lists the ids of sessions that have a workout.
func sessionIDs(sessions []Session) []uuid.UUID {
	var ids []uuid.UUID
	for _, s := range sessions {
		if s.WorkoutID != nil {
			ids = append(ids, s.ID)
		}
	}
	return slices.Clip(ids)
}
