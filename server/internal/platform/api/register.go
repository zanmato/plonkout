// Package api is the HTTP surface every module registers its operations
// through.
//
// One place decides who may call an operation, maps errors to problem
// documents and publishes the declaration in the OpenAPI document, so reading
// a module's registrations tells the whole story of what an endpoint requires.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/userctx"
)

// BasePath is where the API lives. The SPA, OAuth and MCP own the rest.
const BasePath = "/api"

// SessionCookie is the web session cookie. The __Host- prefix pins it to the
// exact origin, https and path /.
const SessionCookie = "__Host-plonkout_sid"

// Security scheme names in the OpenAPI document.
const (
	SchemeSession = "session"
	SchemeBearer  = "bearer"
)

// Op describes one operation.
type Op struct {
	// ID is the operation id, which becomes the generated client's function
	// name. Renaming one is a breaking change for the web client.
	ID string
	// Method and Path, relative to BasePath.
	Method string
	Path   string

	Summary     string
	Description string
	Tags        []string

	// Public marks an operation that needs no signed in user. Everything else
	// requires one, so a forgotten flag is a closed endpoint, never an open one.
	Public bool
	// MCPTool names the MCP tool this operation backs. Empty means a model
	// cannot reach it.
	MCPTool string

	// DefaultStatus overrides the status a successful call answers with.
	DefaultStatus int
	// Errors lists the statuses the handler can produce, for the document.
	Errors []int
}

// Credentials is what a request offers as proof of who it is.
type Credentials struct {
	// SessionToken is the web session cookie value.
	SessionToken string
	// BearerToken is an OAuth access token from the Authorization header.
	BearerToken string
}

// Authenticator resolves credentials to a user. An interface so this package
// does not import the account module, which registers its operations here.
type Authenticator interface {
	Authenticate(ctx context.Context, creds Credentials) (uuid.UUID, error)
}

// Deps is what the registry needs.
type Deps struct {
	API  huma.API
	Auth Authenticator
	// Origin is the public origin. A cookie authenticated write must come from
	// it, which is the CSRF defence: a browser always sends Origin on a cross
	// site POST and a page cannot forge it.
	Origin string
	Logger *slog.Logger
}

// Registry registers operations.
type Registry struct {
	deps       Deps
	registered map[string]Op
}

// NewRegistry builds the registry.
func NewRegistry(deps Deps) *Registry {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	return &Registry{deps: deps, registered: map[string]Op{}}
}

// Huma exposes the underlying API, for the document and the MCP catalog.
func (r *Registry) Huma() huma.API { return r.deps.API }

// Operations returns every registered operation keyed by id.
func (r *Registry) Operations() map[string]Op {
	out := make(map[string]Op, len(r.registered))
	for id, op := range r.registered {
		out[id] = op
	}
	return out
}

// Register declares one operation. It panics on a malformed declaration,
// because registration happens at boot and a broken contract should stop the
// binary rather than serve.
func Register[I, O any](r *Registry, op Op, handler func(context.Context, *I) (*O, error)) {
	switch {
	case op.ID == "":
		panic("api: an operation needs an id")
	case op.Method == "" || op.Path == "":
		panic(fmt.Sprintf("api: operation %q needs a method and a path", op.ID))
	case !strings.HasPrefix(op.Path, "/"):
		panic(fmt.Sprintf("api: the path of operation %q must start with a slash", op.ID))
	case !op.Public && r.deps.Auth == nil:
		panic(fmt.Sprintf("api: operation %q requires a user and the registry has no authenticator", op.ID))
	case op.Public && op.MCPTool != "":
		panic(fmt.Sprintf("api: operation %q is public, and an MCP tool always acts for a user", op.ID))
	}
	if existing, found := r.registered[op.ID]; found {
		panic(fmt.Sprintf("api: the operation id %q is registered twice, for %s %s and %s %s",
			op.ID, existing.Method, existing.Path, op.Method, op.Path))
	}
	r.registered[op.ID] = op

	humaOp := huma.Operation{
		OperationID:   op.ID,
		Method:        op.Method,
		Path:          BasePath + op.Path,
		Summary:       op.Summary,
		Description:   op.Description,
		Tags:          op.Tags,
		DefaultStatus: op.DefaultStatus,
		Errors:        op.Errors,
		Extensions:    map[string]any{},
	}
	if op.Public {
		humaOp.Security = []map[string][]string{}
		humaOp.Extensions["x-public"] = true
	} else {
		humaOp.Security = []map[string][]string{{SchemeSession: {}}, {SchemeBearer: {}}}
		humaOp.Middlewares = huma.Middlewares{r.requireUser}
		if !containsStatus(humaOp.Errors, http.StatusUnauthorized) {
			humaOp.Errors = append(humaOp.Errors, http.StatusUnauthorized)
		}
	}
	if op.MCPTool != "" {
		humaOp.Extensions["x-mcp-tool"] = op.MCPTool
	}

	huma.Register(r.deps.API, humaOp, func(ctx context.Context, in *I) (*O, error) {
		out, err := handler(ctx, in)
		if err != nil {
			problem := MapError(err)
			if problem.Status >= 500 {
				r.deps.Logger.ErrorContext(ctx, "operation failed", "operation", op.ID, "error", err)
			}
			return nil, problem
		}
		return out, nil
	})
}

// requireUser authenticates the request and puts the user on the context, which
// the database hook turns into the row level security setting.
func (r *Registry) requireUser(ctx huma.Context, next func(huma.Context)) {
	creds := Credentials{}
	if bearer, ok := strings.CutPrefix(ctx.Header("Authorization"), "Bearer "); ok {
		creds.BearerToken = strings.TrimSpace(bearer)
	} else if cookie, err := huma.ReadCookie(ctx, SessionCookie); err == nil {
		creds.SessionToken = cookie.Value
		// A cookie rides along on any request the browser makes, so a write
		// must prove it came from our own pages.
		if !safeMethod(ctx.Method()) && ctx.Header("Origin") != r.deps.Origin {
			_ = huma.WriteErr(r.deps.API, ctx, http.StatusForbidden, "cross origin request refused")
			return
		}
	}

	if creds.BearerToken == "" && creds.SessionToken == "" {
		_ = huma.WriteErr(r.deps.API, ctx, http.StatusUnauthorized, "sign in required")
		return
	}

	userID, err := r.deps.Auth.Authenticate(ctx.Context(), creds)
	if err != nil {
		problem := MapError(err)
		if problem.Status >= 500 {
			r.deps.Logger.Error("authentication failed", "error", err)
		}
		_ = huma.WriteErr(r.deps.API, ctx, problem.Status, problem.Detail)
		return
	}

	next(huma.WithContext(ctx, userctx.With(ctx.Context(), userID)))
}

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func containsStatus(statuses []int, status int) bool {
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

// UserID returns the signed in user of an authenticated operation. A handler
// registered without Public can rely on it being present.
func UserID(ctx context.Context) uuid.UUID {
	userID, _ := userctx.From(ctx)
	return userID
}
