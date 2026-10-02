package diary

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Register declares the diary operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"diary"}
	type oneDay struct{ Body Day }

	api.Register(reg, api.Op{
		ID: "get-diary-day", Method: http.MethodGet, Path: "/diary/{day}",
		Summary: "A day of the food diary", Tags: tags, MCPTool: "get_diary",
		Description: "What was eaten per meal with the meal's aim, activities, body weight, and the totals " +
			"against the goal.",
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Day string `path:"day" format:"date" doc:"The user's own calendar day, e.g. 2026-10-02."`
	}) (*oneDay, error) {
		day, err := ParseDay(in.Day)
		if err != nil {
			return nil, err
		}
		d, err := s.Get(ctx, day)
		if err != nil {
			return nil, err
		}
		return &oneDay{Body: d}, nil
	})

	api.Register(reg, api.Op{
		ID: "summarize-diary", Method: http.MethodGet, Path: "/diary",
		Summary: "The food diary over a range of days", Tags: tags, MCPTool: "summarize_diary",
		Description: "Every day from from to to with what was eaten, burned and weighed, and the goal. " +
			"For reviewing how a goal is going.",
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		From string `query:"from" format:"date" required:"true"`
		To   string `query:"to" format:"date" required:"true"`
	}) (*struct{ Body Summary }, error) {
		from, err := ParseDay(in.From)
		if err != nil {
			return nil, err
		}
		to, err := ParseDay(in.To)
		if err != nil {
			return nil, err
		}
		summary, err := s.Summarize(ctx, from, to)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Summary }{Body: summary}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-recent-meals", Method: http.MethodGet, Path: "/diary/{day}/recent",
		Summary: "What the user usually eats, per meal", Tags: tags, MCPTool: "get_recent_meals",
		Description: "From the 60 days before day: each meal's latest earlier logging, whole, and the foods " +
			"eaten at it most often with the amount of the last time. For \"the usual breakfast\" or \"same " +
			"lunch as yesterday\": log those entries with log_food.",
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Day string `path:"day" format:"date" doc:"The user's own calendar day, e.g. 2026-10-02."`
	}) (*struct{ Body Recent }, error) {
		day, err := ParseDay(in.Day)
		if err != nil {
			return nil, err
		}
		recent, err := s.Recent(ctx, day)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Recent }{Body: recent}, nil
	})

	type logBody struct {
		Entries []EntryInput `json:"entries" minItems:"1" maxItems:"50"`
	}
	api.Register(reg, api.Op{
		ID: "log-food", Method: http.MethodPost, Path: "/diary/{day}/entries",
		Summary: "Log what was eaten", Tags: tags, MCPTool: "log_food",
		Description: "Every entry or none. Name each food by foodId or lmvNumber from search_foods and give " +
			"the weight eaten in grams. A one off without a food, such as a restaurant dish, takes a name " +
			"and per100g instead. Answers the day as it now stands.",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Day  string `path:"day" format:"date" doc:"The user's own calendar day, e.g. 2026-10-02."`
		Body logBody
	}) (*oneDay, error) {
		day, err := ParseDay(in.Day)
		if err != nil {
			return nil, err
		}
		if err := s.Log(ctx, day, in.Body.Entries); err != nil {
			return nil, err
		}
		d, err := s.Get(ctx, day)
		if err != nil {
			return nil, err
		}
		return &oneDay{Body: d}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-food-entry", Method: http.MethodPut, Path: "/food-entries/{id}",
		Summary: "Change the amount, meal or day of something eaten", Tags: tags, MCPTool: "update_food_entry",
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body EntryUpdate
	}) (*struct{ Body Entry }, error) {
		entry, err := s.UpdateEntry(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Entry }{Body: entry}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-food-entry", Method: http.MethodDelete, Path: "/food-entries/{id}",
		Summary: "Remove something eaten from the diary", Tags: tags, MCPTool: "delete_food_entry",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.DeleteEntry(ctx, in.ID)
	})

	api.Register(reg, api.Op{
		ID: "log-activity", Method: http.MethodPost, Path: "/diary/{day}/activities",
		Summary: "Log energy burned by exercise", Tags: tags, MCPTool: "log_activity",
		Description: "The app cannot measure it, so it is entered by hand or estimated, e.g. from a logged " +
			"workout's length and intensity and the user's body weight. Say it is an estimate.",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Day  string `path:"day" format:"date" doc:"The user's own calendar day, e.g. 2026-10-02."`
		Body ActivityInput
	}) (*struct{ Body Activity }, error) {
		day, err := ParseDay(in.Day)
		if err != nil {
			return nil, err
		}
		activity, err := s.LogActivity(ctx, day, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Activity }{Body: activity}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-activity", Method: http.MethodDelete, Path: "/activities/{id}",
		Summary: "Remove an activity from the diary", Tags: tags, MCPTool: "delete_activity",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.DeleteActivity(ctx, in.ID)
	})

	type weightBody struct {
		Weight float64 `json:"weight" exclusiveMinimum:"0" maximum:"999" doc:"In the user's weight unit."`
	}
	api.Register(reg, api.Op{
		ID: "log-weight", Method: http.MethodPut, Path: "/diary/{day}/weight",
		Summary: "Set a day's body weight", Tags: tags, MCPTool: "log_weight",
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Day  string `path:"day" format:"date" doc:"The user's own calendar day, e.g. 2026-10-02."`
		Body weightBody
	}) (*struct{}, error) {
		day, err := ParseDay(in.Day)
		if err != nil {
			return nil, err
		}
		return nil, s.SaveWeight(ctx, day, in.Body.Weight)
	})

	api.Register(reg, api.Op{
		ID: "delete-weight", Method: http.MethodDelete, Path: "/diary/{day}/weight",
		Summary: "Remove a day's body weight", Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Day string `path:"day" format:"date" doc:"The user's own calendar day, e.g. 2026-10-02."`
	}) (*struct{}, error) {
		day, err := ParseDay(in.Day)
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteWeight(ctx, day)
	})

	api.Register(reg, api.Op{
		ID: "get-nutrition-goal", Method: http.MethodGet, Path: "/nutrition-goal",
		Summary: "The daily calorie and macro goal", Tags: tags,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body Goal }, error) {
		goal, err := s.Goal(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Goal }{Body: goal}, nil
	})

	api.Register(reg, api.Op{
		ID: "set-nutrition-goal", Method: http.MethodPut, Path: "/nutrition-goal",
		Summary: "Set the daily calorie and macro goal", Tags: tags, MCPTool: "set_nutrition_goal",
		Description: "The budget the diary counts against. Put how it was worked out in notes, so it can be " +
			"revisited as the weight changes.",
	}, func(ctx context.Context, in *struct{ Body GoalInput }) (*struct{ Body Goal }, error) {
		goal, err := s.SaveGoal(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Goal }{Body: goal}, nil
	})
}
