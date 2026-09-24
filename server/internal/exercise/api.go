package exercise

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Register declares the exercise operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"exercises"}
	registerStats(reg, s)

	api.Register(reg, api.Op{
		ID: "list-exercises", Method: http.MethodGet, Path: "/exercises",
		Summary:     "The exercise list",
		Description: "Every exercise, with its muscle group, whether it is trained one arm at a time, and whether it is archived.",
		Tags:        tags, MCPTool: "list_exercises",
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []Exercise }, error) {
		exercises, err := s.List(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body []Exercise }{Body: exercises}, nil
	})

	api.Register(reg, api.Op{
		ID: "create-exercise", Method: http.MethodPost, Path: "/exercises",
		Summary: "Add an exercise",
		Tags:    tags, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusConflict},
	}, func(ctx context.Context, in *struct{ Body ExerciseInput }) (*struct{ Body Exercise }, error) {
		exercise, err := s.Create(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Exercise }{Body: exercise}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-exercise", Method: http.MethodPut, Path: "/exercises/{id}",
		Summary:     "Change an exercise",
		Description: "Renaming also renames it in logged workouts and in the block periodization state.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body ExerciseInput
	}) (*struct{ Body Exercise }, error) {
		exercise, err := s.Update(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Exercise }{Body: exercise}, nil
	})
}
