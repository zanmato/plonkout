// Package system serves the operations about the server itself.
package system

import (
	"context"
	"net/http"
	"time"

	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/system/systemdb"
)

// Health is the answer of the health check.
type Health struct {
	Status   string    `json:"status" example:"ok"`
	Version  string    `json:"version" example:"dev"`
	Database time.Time `json:"database" doc:"The database clock, which proves it answered."`
}

// Register declares the system operations.
func Register(reg *api.Registry, db systemdb.DBTX, version string) {
	queries := systemdb.New(db)

	api.Register(reg, api.Op{
		ID:      "get-health",
		Method:  http.MethodGet,
		Path:    "/health",
		Summary: "Report whether the server and its database are up",
		Tags:    []string{"system"},
		Public:  true,
		Errors:  []int{http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body Health }, error) {
		now, err := queries.DatabaseTime(ctx)
		if err != nil {
			return nil, api.Errorf(http.StatusServiceUnavailable, "database_unavailable", "the database did not answer")
		}
		return &struct{ Body Health }{Body: Health{Status: "ok", Version: version, Database: now}}, nil
	})
}
