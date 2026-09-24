package workout

import (
	"context"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Register declares the workout operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"workouts"}

	api.Register(reg, api.Op{
		ID: "list-workouts", Method: http.MethodGet, Path: "/workouts",
		Summary: "Workouts, newest first",
		Description: "Logged workouts with their exercises and sets, newest first. Narrow with from (inclusive) " +
			"and to (exclusive) dates, an exercise name, and a limit.",
		Tags: tags, MCPTool: "get_workout_history",
	}, func(ctx context.Context, in *struct {
		From     time.Time `query:"from" required:"false" doc:"Only workouts started at or after this time."`
		To       time.Time `query:"to" required:"false" doc:"Only workouts started before this time."`
		Exercise string    `query:"exercise" required:"false" doc:"Only workouts that logged this exercise, by name."`
		Limit    int32     `query:"limit" required:"false" minimum:"1" maximum:"1000" doc:"At most this many workouts."`
	}) (*struct{ Body []Workout }, error) {
		filter := Filter{From: optionalTime(in.From), To: optionalTime(in.To)}
		if in.Exercise != "" {
			filter.Exercise = &in.Exercise
		}
		if in.Limit > 0 {
			filter.Limit = &in.Limit
		}
		workouts, err := s.List(ctx, filter)
		if err != nil {
			return nil, err
		}
		return &struct{ Body []Workout }{Body: workouts}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-latest-workout", Method: http.MethodGet, Path: "/workouts/latest",
		Summary:     "The most recent workout with a name",
		Description: "Used to suggest the exercises of the last workout with the same name.",
		Tags:        tags, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		Name      string    `query:"name" minLength:"1"`
		ExcludeID uuid.UUID `query:"excludeId" required:"false" doc:"Skip this workout, usually the one being edited."`
	}) (*struct{ Body Workout }, error) {
		var exclude *uuid.UUID
		if in.ExcludeID != uuid.Nil {
			exclude = &in.ExcludeID
		}
		workout, err := s.Latest(ctx, in.Name, exclude)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Workout }{Body: workout}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-workout", Method: http.MethodGet, Path: "/workouts/{id}",
		Summary: "One workout",
		Tags:    tags, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{ Body Workout }, error) {
		workout, err := s.Get(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Workout }{Body: workout}, nil
	})

	api.Register(reg, api.Op{
		ID: "create-workout", Method: http.MethodPost, Path: "/workouts",
		Summary: "Log a new workout",
		Tags:    tags, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body WorkoutInput }) (*struct{ Body Workout }, error) {
		workout, err := s.Create(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Workout }{Body: workout}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-workout", Method: http.MethodPut, Path: "/workouts/{id}",
		Summary:     "Replace a workout",
		Description: "The body carries the revision the client last read. A newer revision on the server answers 409 stale_revision.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body WorkoutUpdate
	}) (*struct{ Body Workout }, error) {
		workout, err := s.Update(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Workout }{Body: workout}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-workout", Method: http.MethodDelete, Path: "/workouts/{id}",
		Summary: "Delete a workout",
		Tags:    tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.Delete(ctx, in.ID)
	})
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
