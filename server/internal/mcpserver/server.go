// Package mcpserver lets an assistant such as Claude read a user's training
// and plan it, over the Model Context Protocol.
//
// A tool call is an HTTP request through the same handler the app uses, made
// with the caller's own access token. Authentication, row level security,
// validation and error mapping all apply unchanged, so an assistant can reach
// exactly what its user could reach and nothing more. Each tool is declared on
// the operation that backs it (api.Op.MCPTool), and its input schema is read
// from the live OpenAPI document, so the two cannot disagree.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zanmato/plonkout/server/internal/oauth"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

const tokenKey = "plonkout.token"

// Deps is what the MCP server needs.
type Deps struct {
	// Handler serves the API. Tool calls are requests through it.
	Handler  http.Handler
	Registry *api.Registry
	Tokens   *oauth.Service
	Version  string
	Logger   *slog.Logger
}

// Server serves MCP over streamable HTTP.
type Server struct {
	deps  Deps
	tools []tool
}

type tool struct {
	def *mcp.Tool
	op  huma.Operation
}

// New builds the tool catalog from the registered operations.
func New(deps Deps) (*Server, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	doc := deps.Registry.Huma().OpenAPI()

	var tools []tool
	for id, op := range deps.Registry.Operations() {
		if op.MCPTool == "" {
			continue
		}
		operation := findOperation(doc, id)
		if operation == nil {
			return nil, fmt.Errorf("mcp: operation %q is not in the OpenAPI document", id)
		}
		schema, err := inputSchema(doc, operation)
		if err != nil {
			return nil, fmt.Errorf("mcp: tool %s: %w", op.MCPTool, err)
		}
		description := op.Summary
		if op.Description != "" {
			description += ". " + op.Description
		}
		readOnly := op.Method == http.MethodGet
		annotations := &mcp.ToolAnnotations{ReadOnlyHint: readOnly, Title: op.Summary}
		if !readOnly {
			destructive := op.MCPTool == "set_session_status"
			annotations.DestructiveHint = &destructive
		}
		tools = append(tools, tool{
			def: &mcp.Tool{
				Name:        op.MCPTool,
				Title:       op.Summary,
				Description: description,
				InputSchema: schema,
				Annotations: annotations,
			},
			op: *operation,
		})
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].def.Name < tools[j].def.Name })
	return &Server{deps: deps, tools: tools}, nil
}

// Tools lists the tool definitions, for tests and diagnostics.
func (s *Server) Tools() []*mcp.Tool {
	out := make([]*mcp.Tool, len(s.tools))
	for i, t := range s.tools {
		out[i] = t.def
	}
	return out
}

// Handler serves /mcp. A request without a valid token for this resource is
// answered 401 with a WWW-Authenticate header pointing at the protected
// resource metadata, which is how a client discovers where to sign in.
func (s *Server) Handler() http.Handler {
	transport := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.server()
	}, &mcp.StreamableHTTPOptions{Stateless: true, Logger: s.deps.Logger})

	authenticated := sdkauth.RequireBearerToken(s.verify, &sdkauth.RequireBearerTokenOptions{
		ResourceMetadataURL: s.deps.Tokens.ResourceMetadataURL(),
		Scopes:              []string{oauth.ScopeTraining},
	})(transport)
	return cors(authenticated)
}

func (s *Server) verify(ctx context.Context, token string, _ *http.Request) (*sdkauth.TokenInfo, error) {
	info, err := s.deps.Tokens.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", sdkauth.ErrInvalidToken, err)
	}
	return &sdkauth.TokenInfo{
		UserID:     info.UserID.String(),
		Scopes:     info.Scopes,
		Expiration: info.Expires,
		Extra:      map[string]any{tokenKey: token},
	}, nil
}

// server builds the MCP server for one request. Stateless mode keeps nothing
// between requests, which suits a server with one instance and no sessions.
func (s *Server) server() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "plonkout",
		Title:   "Plonkout",
		Version: s.deps.Version,
	}, &mcp.ServerOptions{Instructions: instructions})

	for _, t := range s.tools {
		server.AddTool(t.def, s.handler(t))
	}
	addPrompts(server)
	return server
}

func (s *Server) handler(t tool) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := sdkauth.TokenInfoFromContext(ctx)
		token, _ := info.Extra[tokenKey].(string)
		if info == nil || token == "" {
			return toolError("unauthorized", "the call carries no access token"), nil
		}

		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return toolError("invalid_arguments", "the arguments are not a JSON object"), nil
			}
		}
		httpReq, err := buildRequest(ctx, t.op, args)
		if err != nil {
			return toolError("invalid_arguments", err.Error()), nil
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)

		rec := &recorder{header: http.Header{}}
		s.deps.Handler.ServeHTTP(rec, httpReq)

		if rec.status() >= 300 {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: rec.body.String()}},
			}, nil
		}
		text := rec.body.String()
		if text == "" {
			text = `{"ok":true}`
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
	}
}

// buildRequest turns tool arguments into the operation's request: path and
// query parameters by name, everything else into the JSON body.
func buildRequest(ctx context.Context, op huma.Operation, args map[string]any) (*http.Request, error) {
	path := op.Path
	query := url.Values{}
	body := map[string]any{}
	for key, value := range args {
		body[key] = value
	}

	for _, param := range op.Parameters {
		value, ok := args[param.Name]
		delete(body, param.Name)
		if !ok || value == nil {
			if param.In == "path" {
				return nil, fmt.Errorf("%s is required", param.Name)
			}
			continue
		}
		text := fmt.Sprint(value)
		switch param.In {
		case "path":
			path = strings.ReplaceAll(path, "{"+param.Name+"}", url.PathEscape(text))
		case "query":
			query.Set(param.Name, text)
		}
	}

	var payload []byte
	if op.RequestBody != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = encoded
	} else if len(body) > 0 {
		keys := make([]string, 0, len(body))
		for key := range body {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("unknown arguments: %s", strings.Join(keys, ", "))
	}

	target := &url.URL{Path: path, RawQuery: query.Encode()}
	req, err := http.NewRequestWithContext(ctx, op.Method, target.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func toolError(code, detail string) *mcp.CallToolResult {
	body, _ := json.Marshal(map[string]string{"code": code, "detail": detail})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}
}

// cors lets browser based MCP clients, the MCP Inspector among them, connect.
// Tokens never ride along on their own like cookies do, so any origin may.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Session-Id, MCP-Protocol-Version, Last-Event-ID")
		h.Set("Access-Control-Expose-Headers", "Mcp-Session-Id, WWW-Authenticate")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type recorder struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header { return r.header }

func (r *recorder) Write(p []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	return r.body.Write(p)
}

func (r *recorder) WriteHeader(code int) {
	if r.code == 0 {
		r.code = code
	}
}

func (r *recorder) status() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}
