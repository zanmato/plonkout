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
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/web"
	"github.com/zanmato/plonkout/server/internal/system"
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

	humaAPI := humago.New(mux, humaConfig)
	reg := api.NewRegistry(api.Deps{
		API:    humaAPI,
		Origin: cfg.Server.BaseURL,
		Logger: deps.Logger,
	})

	system.Register(reg, deps.Pool, Version)

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
