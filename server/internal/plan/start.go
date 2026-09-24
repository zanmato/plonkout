package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/zanmato/plonkout/server/internal/plan/plandb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/workout"
)

// Start begins a session: it logs a workout prefilled from the session's
// targets and links the two. Starting a session that already has a workout
// answers with that workout, so a double tap does not log twice.
func (s *Service) Start(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var workoutID uuid.UUID
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		locked, err := q.GetSessionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no such session", api.ErrNotFound)
		} else if err != nil {
			return err
		}

		row, err := q.GetSession(ctx, id)
		if err != nil {
			return err
		}
		if row.WorkoutID != nil {
			workoutID = *row.WorkoutID
			return nil
		}
		if locked.Status == StatusSkipped {
			return api.Errorf(http.StatusConflict, "session_skipped", "the session was skipped, put it back before starting it")
		}

		sessions, err := loadSessions(ctx, q, []plandb.ListSessionsRow{plandb.ListSessionsRow(row)}, false)
		if err != nil {
			return err
		}
		plan, err := q.GetPlan(ctx, row.PlanID)
		if err != nil {
			return err
		}
		dominant, err := dominantArm(ctx, q)
		if err != nil {
			return err
		}

		created, err := workout.CreateInTx(ctx, tx, prefill(plan.Name, sessions[0], dominant, time.Now()), workout.CreateOptions{
			PlannedSessionID: &id,
		})
		if err != nil {
			return err
		}
		workoutID = created.ID
		return q.SetSessionStatus(ctx, plandb.SetSessionStatusParams{ID: id, Status: StatusInProgress})
	})
	return workoutID, err
}

// prefill turns a session into a workout. Every set of every target becomes
// a logged set linked to it, with nothing entered yet: the app shows the
// target as a placeholder until the set is done. A single arm exercise gets a
// set for each arm, the off arm's at its share of the weight.
func prefill(planName string, session Session, dominant string, started time.Time) workout.WorkoutInput {
	offArm := "left"
	if dominant == "left" {
		offArm = "right"
	}

	name := planName
	if session.Label != "" {
		name = strings.TrimSpace(planName + " " + session.Label)
	}
	in := workout.WorkoutInput{
		Name:      name,
		Started:   started,
		Notes:     session.Notes,
		Exercises: make([]workout.WorkoutExercise, 0, len(session.Exercises)),
	}

	for _, planned := range session.Exercises {
		exerciseID, plannedID := planned.ExerciseID, planned.ID
		exercise := workout.WorkoutExercise{
			ExerciseID:        &exerciseID,
			Name:              planned.Exercise,
			MuscleGroup:       planned.MuscleGroup,
			Type:              planned.Type,
			DisplayType:       planned.DisplayType,
			SingleArm:         planned.SingleArm,
			Intensity:         session.Intensity,
			PlannedExerciseID: &plannedID,
			Sets:              []workout.WorkoutSet{},
		}
		for _, target := range planned.Targets {
			targetID := target.ID
			for seq := int32(1); seq <= target.Sets; seq++ {
				if !planned.SingleArm {
					exercise.Sets = append(exercise.Sets, prescribed(targetID, seq, ""))
					continue
				}
				exercise.Sets = append(exercise.Sets,
					prescribed(targetID, seq, dominant),
					prescribed(targetID, seq, offArm))
			}
		}
		in.Exercises = append(in.Exercises, exercise)
	}
	return in
}

func prescribed(targetID uuid.UUID, seq int32, arm string) workout.WorkoutSet {
	return workout.WorkoutSet{Type: "regular", Arm: arm, TargetID: &targetID, TargetSeq: &seq}
}

// OffArmWeight is the weight the non dominant arm works with, rounded to a
// half, which is what plates and stacks allow.
func OffArmWeight(weight, percent float64) float64 {
	return math.Round(weight*percent/100*2) / 2
}

// dominantArm reads the user's dominant arm, right unless they said otherwise.
func dominantArm(ctx context.Context, q *plandb.Queries) (string, error) {
	raw, err := q.GetSettingValue(ctx, "dominantArm")
	if errors.Is(err, pgx.ErrNoRows) {
		return "right", nil
	} else if err != nil {
		return "", err
	}
	var arm string
	if json.Unmarshal(raw, &arm) != nil || (arm != "left" && arm != "right") {
		return "right", nil
	}
	return arm, nil
}
