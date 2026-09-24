// Package setting stores a user's preferences as key and JSON value.
//
// Preferences that only matter on one device, the theme and the language,
// stay in the browser. Everything here is visible to MCP clients too, which is
// why the weight unit lives on the server.
package setting

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/setting/settingdb"
)

// Service is the settings module.
type Service struct {
	q *settingdb.Queries
}

// NewService builds the settings module.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{q: settingdb.New(pool)}
}

// All returns every setting.
func (s *Service) All(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := s.q.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]json.RawMessage, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Value
	}
	return out, nil
}

// Put stores one setting.
func (s *Service) Put(ctx context.Context, key string, value json.RawMessage) error {
	return s.q.PutSetting(ctx, settingdb.PutSettingParams{Key: key, Value: value})
}

// Register declares the settings operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"settings"}

	api.Register(reg, api.Op{
		ID: "list-settings", Method: http.MethodGet, Path: "/settings",
		Summary: "Every setting, as key and JSON value", Tags: tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body map[string]json.RawMessage }, error) {
		settings, err := s.All(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body map[string]json.RawMessage }{Body: settings}, nil
	})

	type putBody struct {
		Value json.RawMessage `json:"value"`
	}
	api.Register(reg, api.Op{
		ID: "put-setting", Method: http.MethodPut, Path: "/settings/{key}",
		Summary: "Store a setting", Tags: tags, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *struct {
		Key  string `path:"key" pattern:"^[A-Za-z0-9_.-]{1,64}$"`
		Body putBody
	}) (*struct{}, error) {
		if len(in.Body.Value) == 0 {
			return nil, api.Errorf(http.StatusUnprocessableEntity, "invalid_request", "a value is required")
		}
		return nil, s.Put(ctx, in.Key, in.Body.Value)
	})
}
