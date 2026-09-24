// Package server assembles the HTTP surface: the API every module registers
// into, the SPA, and the headers every response carries.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/account"
	"github.com/zanmato/plonkout/server/internal/exercise"
	"github.com/zanmato/plonkout/server/internal/importer"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/web"
	"github.com/zanmato/plonkout/server/internal/setting"
	"github.com/zanmato/plonkout/server/internal/system"
	"github.com/zanmato/plonkout/server/internal/template"
	"github.com/zanmato/plonkout/server/internal/workout"
)

// Version is set at build time.
var Version = "dev"

// Server is the assembled application.
type Server struct {
	api     huma.API
	handler http.Handler
}

// Deps is what the server is built from.
type Deps struct {
	Config *config.Config
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

func init() {
	api.UseProblemErrors()
	// Handlers never answer with a nil slice, so an array is an array. A
	// nullable one would make every list in the web client T[] | null.
	huma.DefaultArrayNullable = false
}

// New builds the server. It never contacts the database, so the openapi
// subcommand can build the real thing from a pool that is never used.
func New(deps Deps) (*Server, error) {
	cfg := deps.Config
	mux := http.NewServeMux()

	humaConfig := huma.DefaultConfig("Plonkout API", Version)
	humaConfig.Info.Description = "Workout logging and training plans."
	humaConfig.Servers = []*huma.Server{{URL: "/"}}
	// The $schema link Huma adds to every response is noise for the generated
	// client, and the schemas endpoint it points at is not served.
	humaConfig.CreateHooks = nil
	humaConfig.SchemasPath = ""
	humaConfig.OpenAPIPath = api.BasePath + "/openapi"
	humaConfig.DocsPath = ""
	if cfg.Development {
		humaConfig.DocsPath = api.BasePath + "/docs"
	}
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		api.SchemeSession: {
			Type:        "apiKey",
			In:          "cookie",
			Name:        api.SessionCookie,
			Description: "The web session cookie set by signing in with a passkey.",
		},
		api.SchemeBearer: {
			Type:        "http",
			Scheme:      "bearer",
			Description: "An OAuth access token, as issued to MCP clients.",
		},
	}

	accounts, err := account.NewService(deps.Pool, cfg)
	if err != nil {
		return nil, err
	}

	humaAPI := humago.New(mux, humaConfig)
	reg := api.NewRegistry(api.Deps{
		API:    humaAPI,
		Auth:   accounts,
		Origin: cfg.Server.BaseURL,
		Logger: deps.Logger,
	})

	system.Register(reg, deps.Pool, Version)
	account.Register(reg, accounts)
	exercise.Register(reg, exercise.NewService(deps.Pool))
	workout.Register(reg, workout.NewService(deps.Pool))
	template.Register(reg, template.NewService(deps.Pool))
	setting.Register(reg, setting.NewService(deps.Pool))
	importer.Register(reg, importer.NewService(deps.Pool))
	api.FixNullableEnums(humaAPI.OpenAPI())

	if cfg.Frontend.Path != "" {
		mux.Handle("/", web.SPA(cfg.Frontend.Path))
	}

	return &Server{api: humaAPI, handler: securityHeaders(mux)}, nil
}

// Handler is what the listener serves.
func (s *Server) Handler() http.Handler { return s.handler }

// OpenAPI returns the document, indented so a committed copy diffs cleanly.
func (s *Server) OpenAPI() ([]byte, error) {
	raw, err := json.Marshal(s.api.OpenAPI())
	if err != nil {
		return nil, fmt.Errorf("marshal the OpenAPI document: %w", err)
	}
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return nil, fmt.Errorf("indent the OpenAPI document: %w", err)
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
