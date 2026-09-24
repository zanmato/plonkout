// Package overview answers "where is this person at": the first thing an
// assistant asks before planning or reviewing training.
package overview

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/overview/overviewdb"
	"github.com/zanmato/plonkout/server/internal/plan"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Context is the overview.
type Context struct {
	Username      string          `json:"username"`
	Now           time.Time       `json:"now" doc:"The server's clock, to reason about dates from."`
	WeightUnit    string          `json:"weightUnit" enum:"kg,lbs" doc:"Every weight in this account is in this unit."`
	DistanceUnit  string          `json:"distanceUnit"`
	DominantArm   string          `json:"dominantArm" enum:"left,right" doc:"The arm that gets the listed weight on single arm exercises."`
	TotalWorkouts int64           `json:"totalWorkouts"`
	ActivePlans   []PlanSummary   `json:"activePlans"`
	Recent        []RecentWorkout `json:"recentWorkouts" doc:"The latest workouts, newest first."`
}

// PlanSummary is an active plan in brief.
type PlanSummary struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	Goal      string        `json:"goal"`
	Sessions  int           `json:"sessions"`
	Completed int           `json:"completed"`
	Skipped   int           `json:"skipped"`
	Pending   int           `json:"pending"`
	Next      *plan.Session `json:"next" doc:"The session the user will pick next, with its targets."`
}

// RecentWorkout is a logged workout in brief.
type RecentWorkout struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Started          time.Time  `json:"started"`
	Ended            *time.Time `json:"ended"`
	PlannedSessionID *uuid.UUID `json:"plannedSessionId,omitempty"`
	Exercises        []string   `json:"exercises"`
}

// Register declares the overview operation.
func Register(reg *api.Registry, pool *pgxpool.Pool, plans *plan.Service) {
	q := overviewdb.New(pool)

	api.Register(reg, api.Op{
		ID: "get-context", Method: http.MethodGet, Path: "/context",
		Summary: "Where the user is at",
		Description: "Units, dominant arm, active plans with progress and the next session, and the latest " +
			"workouts. Read this first: every weight elsewhere is in weightUnit.",
		Tags: []string{"overview"}, MCPTool: "get_context",
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body Context }, error) {
		out := Context{
			Now: time.Now(), WeightUnit: "kg", DistanceUnit: "km", DominantArm: "right",
			ActivePlans: []PlanSummary{}, Recent: []RecentWorkout{},
		}
		var err error
		if out.Username, err = q.Username(ctx, api.UserID(ctx)); err != nil {
			return nil, err
		}

		settings, err := q.Settings(ctx, []string{"weightUnit", "distanceUnit", "dominantArm"})
		if err != nil {
			return nil, err
		}
		for _, setting := range settings {
			var value string
			if json.Unmarshal(setting.Value, &value) != nil || value == "" {
				continue
			}
			switch setting.Key {
			case "weightUnit":
				out.WeightUnit = value
			case "distanceUnit":
				out.DistanceUnit = value
			case "dominantArm":
				out.DominantArm = value
			}
		}

		if out.TotalWorkouts, err = q.CountWorkouts(ctx); err != nil {
			return nil, err
		}
		recent, err := q.RecentWorkouts(ctx, 5)
		if err != nil {
			return nil, err
		}
		for _, w := range recent {
			out.Recent = append(out.Recent, RecentWorkout{
				ID: w.ID, Name: w.Name, Started: w.Started, Ended: w.Ended,
				PlannedSessionID: w.PlannedSessionID, Exercises: w.Exercises,
			})
		}

		active := "active"
		plans, err := plans.List(ctx, &active)
		if err != nil {
			return nil, err
		}
		for _, p := range plans {
			summary := PlanSummary{ID: p.ID, Name: p.Name, Goal: p.Goal, Sessions: len(p.Sessions)}
			for i := range p.Sessions {
				session := p.Sessions[i]
				switch session.Status {
				case plan.StatusCompleted:
					summary.Completed++
				case plan.StatusSkipped:
					summary.Skipped++
				default:
					summary.Pending++
					if summary.Next == nil {
						summary.Next = &session
					}
				}
			}
			out.ActivePlans = append(out.ActivePlans, summary)
		}
		return &struct{ Body Context }{Body: out}, nil
	})
}
