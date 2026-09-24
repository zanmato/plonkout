// Package template stores workout templates. A template is only ever read
// and written whole, so it is kept as a document.
package template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/template/templatedb"
	"github.com/zanmato/plonkout/server/internal/workout"
)

// Template is a reusable workout outline.
type Template struct {
	ID        uuid.UUID                 `json:"id"`
	Name      string                    `json:"name"`
	Notes     string                    `json:"notes"`
	Exercises []workout.WorkoutExercise `json:"exercises"`
	Created   time.Time                 `json:"created"`
	Updated   time.Time                 `json:"updated"`
}

// TemplateInput is what a client writes.
type TemplateInput struct {
	Name      string                    `json:"name" maxLength:"200"`
	Notes     string                    `json:"notes" maxLength:"5000"`
	Exercises []workout.WorkoutExercise `json:"exercises" maxItems:"50"`
}

type document struct {
	Notes     string                    `json:"notes"`
	Exercises []workout.WorkoutExercise `json:"exercises"`
}

// Service is the template module.
type Service struct {
	q *templatedb.Queries
}

// NewService builds the template module.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{q: templatedb.New(pool)}
}

func (s *Service) List(ctx context.Context) ([]Template, error) {
	rows, err := s.q.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Template, len(rows))
	for i, row := range rows {
		if out[i], err = toTemplate(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Template, error) {
	row, err := s.q.GetTemplate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, fmt.Errorf("%w: no such template", api.ErrNotFound)
	} else if err != nil {
		return Template{}, err
	}
	return toTemplate(row)
}

func (s *Service) Create(ctx context.Context, in TemplateInput) (Template, error) {
	doc, err := encode(in)
	if err != nil {
		return Template{}, err
	}
	row, err := s.q.CreateTemplate(ctx, templatedb.CreateTemplateParams{Name: in.Name, Doc: doc})
	if err != nil {
		return Template{}, err
	}
	return toTemplate(row)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in TemplateInput) (Template, error) {
	doc, err := encode(in)
	if err != nil {
		return Template{}, err
	}
	row, err := s.q.UpdateTemplate(ctx, templatedb.UpdateTemplateParams{ID: id, Name: in.Name, Doc: doc})
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, fmt.Errorf("%w: no such template", api.ErrNotFound)
	} else if err != nil {
		return Template{}, err
	}
	return toTemplate(row)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.q.DeleteTemplate(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("%w: no such template", api.ErrNotFound)
	}
	return nil
}

func encode(in TemplateInput) ([]byte, error) {
	exercises := in.Exercises
	if exercises == nil {
		exercises = []workout.WorkoutExercise{}
	}
	return json.Marshal(document{Notes: in.Notes, Exercises: exercises})
}

func toTemplate(row templatedb.Template) (Template, error) {
	var doc document
	if err := json.Unmarshal(row.Doc, &doc); err != nil {
		return Template{}, fmt.Errorf("read template %s: %w", row.ID, err)
	}
	if doc.Exercises == nil {
		doc.Exercises = []workout.WorkoutExercise{}
	}
	return Template{
		ID: row.ID, Name: row.Name, Notes: doc.Notes, Exercises: doc.Exercises,
		Created: row.CreatedAt, Updated: row.UpdatedAt,
	}, nil
}

// Register declares the template operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"templates"}
	type one struct{ Body Template }

	api.Register(reg, api.Op{
		ID: "list-templates", Method: http.MethodGet, Path: "/templates",
		Summary: "Workout templates, newest first", Tags: tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []Template }, error) {
		templates, err := s.List(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body []Template }{Body: templates}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-template", Method: http.MethodGet, Path: "/templates/{id}",
		Summary: "One template", Tags: tags, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*one, error) {
		t, err := s.Get(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &one{Body: t}, nil
	})

	api.Register(reg, api.Op{
		ID: "create-template", Method: http.MethodPost, Path: "/templates",
		Summary: "Save a template", Tags: tags, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct{ Body TemplateInput }) (*one, error) {
		t, err := s.Create(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &one{Body: t}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-template", Method: http.MethodPut, Path: "/templates/{id}",
		Summary: "Replace a template", Tags: tags, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body TemplateInput
	}) (*one, error) {
		t, err := s.Update(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &one{Body: t}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-template", Method: http.MethodDelete, Path: "/templates/{id}",
		Summary: "Delete a template", Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.Delete(ctx, in.ID)
	})
}
