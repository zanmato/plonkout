package plan

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Register declares the plan operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"plans"}
	type onePlan struct{ Body Plan }
	type oneSession struct{ Body Session }

	api.Register(reg, api.Op{
		ID: "list-plans", Method: http.MethodGet, Path: "/plans",
		Summary: "Training plans, newest first", Tags: tags, MCPTool: "list_plans",
		Description: "Every plan with its sessions, exercises and targets, optionally only those with a status.",
	}, func(ctx context.Context, in *struct {
		Status string `query:"status" enum:"active,completed,archived" required:"false"`
	}) (*struct{ Body []Plan }, error) {
		var status *string
		if in.Status != "" {
			status = &in.Status
		}
		plans, err := s.List(ctx, status)
		if err != nil {
			return nil, err
		}
		return &struct{ Body []Plan }{Body: plans}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-plan", Method: http.MethodGet, Path: "/plans/{id}",
		Summary: "One plan", Tags: tags, MCPTool: "get_plan",
		Description: "With actuals, every session that has a workout carries what was planned against what was " +
			"logged: sets per target, whether the target was met, the top RPE and the best estimated 1RM.",
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID      uuid.UUID `path:"id"`
		Actuals bool      `query:"actuals" required:"false"`
	}) (*onePlan, error) {
		p, err := s.Get(ctx, in.ID, in.Actuals)
		if err != nil {
			return nil, err
		}
		return &onePlan{Body: p}, nil
	})

	api.Register(reg, api.Op{
		ID: "create-plan", Method: http.MethodPost, Path: "/plans",
		Summary: "Create a plan with its queue of sessions", Tags: tags, MCPTool: "create_plan",
		Description: "Sessions are queued in the order given. Exercises are named as in the exercise list. " +
			"An unknown name is refused with suggestions unless createMissingExercises is set.",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body PlanInput }) (*onePlan, error) {
		p, err := s.Create(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &onePlan{Body: p}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-plan", Method: http.MethodPut, Path: "/plans/{id}",
		Summary: "Change a plan's name, goal, notes, status or start date", Tags: tags, MCPTool: "update_plan",
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body PlanUpdate
	}) (*onePlan, error) {
		p, err := s.Update(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &onePlan{Body: p}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-plan", Method: http.MethodDelete, Path: "/plans/{id}",
		Summary:     "Delete a plan",
		Description: "Workouts logged for its sessions stay, without the link to the plan.",
		Tags:        tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.Delete(ctx, in.ID)
	})

	type addSessionsBody struct {
		Sessions               []SessionInput `json:"sessions" minItems:"1" maxItems:"200"`
		AfterSessionID         *uuid.UUID     `json:"afterSessionId,omitempty" doc:"Insert after this session. Omitted appends to the end of the queue."`
		CreateMissingExercises bool           `json:"createMissingExercises,omitempty"`
	}
	api.Register(reg, api.Op{
		ID: "add-sessions", Method: http.MethodPost, Path: "/plans/{id}/sessions",
		Summary: "Add sessions to a plan's queue", Tags: tags, MCPTool: "add_sessions",
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body addSessionsBody
	}) (*onePlan, error) {
		p, err := s.AddSessions(ctx, in.ID, in.Body.AfterSessionID, in.Body.Sessions, in.Body.CreateMissingExercises)
		if err != nil {
			return nil, err
		}
		return &onePlan{Body: p}, nil
	})

	type reorderBody struct {
		SessionIDs []uuid.UUID `json:"sessionIds" doc:"Every pending session of the plan, in the new order."`
	}
	api.Register(reg, api.Op{
		ID: "reorder-sessions", Method: http.MethodPut, Path: "/plans/{id}/order",
		Summary:     "Reorder a plan's pending sessions",
		Description: "Sessions already started, completed or skipped keep their place.",
		Tags:        tags, MCPTool: "reorder_sessions",
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body reorderBody
	}) (*onePlan, error) {
		p, err := s.Reorder(ctx, in.ID, in.Body.SessionIDs)
		if err != nil {
			return nil, err
		}
		return &onePlan{Body: p}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-planned-session", Method: http.MethodGet, Path: "/planned-sessions/{id}",
		Summary: "One planned session with its exercises and targets", Tags: tags,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*oneSession, error) {
		row, err := s.q.GetSession(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		session, err := s.session(ctx, row.PlanID, in.ID)
		if err != nil {
			return nil, err
		}
		return &oneSession{Body: session}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-session", Method: http.MethodPatch, Path: "/planned-sessions/{id}",
		Summary:     "Change a pending session",
		Description: "Fields left out are kept. exercises, when given, replaces every planned exercise and target.",
		Tags:        tags, MCPTool: "update_session",
		Errors: []int{http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body SessionUpdate
	}) (*oneSession, error) {
		session, err := s.UpdateSession(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &oneSession{Body: session}, nil
	})

	type statusBody struct {
		Status string `json:"status" enum:"skipped,pending,deleted" doc:"skipped skips a pending session, pending puts a skipped one back, deleted removes one that was never started."`
		Reason string `json:"reason,omitempty" maxLength:"500"`
	}
	type statusOutput struct {
		Body struct {
			Session *Session `json:"session,omitempty" doc:"Absent when the session was deleted."`
		}
	}
	api.Register(reg, api.Op{
		ID: "set-session-status", Method: http.MethodPost, Path: "/planned-sessions/{id}/status",
		Summary: "Skip, unskip or delete a session", Tags: tags, MCPTool: "set_session_status",
		Errors: []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body statusBody
	}) (*statusOutput, error) {
		session, err := s.SetStatus(ctx, in.ID, in.Body.Status, in.Body.Reason)
		if err != nil {
			return nil, err
		}
		out := &statusOutput{}
		out.Body.Session = session
		return out, nil
	})

	type startOutput struct {
		Body struct {
			WorkoutID uuid.UUID `json:"workoutId"`
		}
	}
	api.Register(reg, api.Op{
		ID: "start-session", Method: http.MethodPost, Path: "/planned-sessions/{id}/start",
		Summary: "Start a session", Tags: tags,
		Description: "Logs a workout prefilled with the session's targets and links it to the session. " +
			"A session that already has a workout answers with that workout.",
		Errors: []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*startOutput, error) {
		workoutID, err := s.Start(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		out := &startOutput{}
		out.Body.WorkoutID = workoutID
		return out, nil
	})

	api.Register(reg, api.Op{
		ID: "get-session-comparison", Method: http.MethodGet, Path: "/planned-sessions/{id}/comparison",
		Summary: "What a session planned against what was logged", Tags: tags,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{ Body Comparison }, error) {
		comparison, err := s.Comparison(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Comparison }{Body: comparison}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-queue", Method: http.MethodGet, Path: "/queue",
		Summary: "Sessions still to do in active plans, in order", Tags: tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []QueueEntry }, error) {
		queue, err := s.Queue(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body []QueueEntry }{Body: queue}, nil
	})
}
