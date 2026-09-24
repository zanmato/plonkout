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
		Summary:     "Workouts, newest first",
		Description: "Every workout with its exercises and sets, optionally between from (inclusive) and to (exclusive).",
		Tags:        tags,
	}, func(ctx context.Context, in *struct {
		From time.Time `query:"from" required:"false"`
		To   time.Time `query:"to" required:"false"`
	}) (*struct{ Body []Workout }, error) {
		workouts, err := s.List(ctx, optionalTime(in.From), optionalTime(in.To))
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
