package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Mount serves the endpoints OAuth clients talk to directly. They follow the
// RFCs rather than the API's conventions: form bodies, snake_case JSON and
// error objects of their own.
func (s *Service) Mount(mux *http.ServeMux, logger *slog.Logger) {
	h := &handlers{s: s, logger: logger}

	for _, path := range []string{"/.well-known/oauth-authorization-server", "/.well-known/openid-configuration"} {
		mux.Handle("GET "+path, cors(http.HandlerFunc(h.serverMetadata)))
	}
	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		mux.Handle("GET "+path, cors(http.HandlerFunc(h.resourceMetadata)))
	}
	mux.Handle("POST /oauth/register", cors(http.HandlerFunc(h.register)))
	mux.Handle("POST /oauth/token", cors(http.HandlerFunc(h.token)))
	mux.Handle("POST /oauth/revoke", cors(http.HandlerFunc(h.revoke)))
	mux.HandleFunc("GET /oauth/authorize", h.authorize)
	for _, path := range []string{"/.well-known/", "/oauth/"} {
		mux.Handle("OPTIONS "+path, cors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})))
	}
}

// ResourceMetadataURL is where the MCP endpoint's 401 points clients.
func (s *Service) ResourceMetadataURL() string {
	return s.cfg.Issuer + "/.well-known/oauth-protected-resource/mcp"
}

type handlers struct {
	s      *Service
	logger *slog.Logger
}

// cors lets browser based MCP clients, the MCP Inspector among them, reach the
// discovery and token endpoints. None of them use cookies, so any origin may.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
		next.ServeHTTP(w, r)
	})
}

func (h *handlers) serverMetadata(w http.ResponseWriter, _ *http.Request) {
	issuer := h.s.cfg.Issuer
	doc := map[string]any{
		"issuer":                                         issuer,
		"authorization_endpoint":                         issuer + "/oauth/authorize",
		"token_endpoint":                                 issuer + "/oauth/token",
		"revocation_endpoint":                            issuer + "/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"scopes_supported":                               SupportedScopes,
		"client_id_metadata_document_supported":          true,
		"authorization_response_iss_parameter_supported": true,
	}
	if h.s.cfg.DynamicRegistration {
		doc["registration_endpoint"] = issuer + "/oauth/register"
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *handlers) resourceMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 h.s.cfg.Resource,
		"authorization_servers":    []string{h.s.cfg.Issuer},
		"scopes_supported":         SupportedScopes,
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Plonkout",
	})
}

func (h *handlers) register(w http.ResponseWriter, r *http.Request) {
	var req RegistrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "the body is not a JSON registration")
		return
	}
	resp, err := h.s.Register(r.Context(), req)
	if err != nil {
		if statusFor(err) >= 500 {
			h.logger.Error("client registration failed", "error", err)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
			return
		}
		code := "invalid_client_metadata"
		if errors.Is(err, errRedirectURI) {
			code = "invalid_redirect_uri"
		}
		writeOAuthError(w, http.StatusBadRequest, code, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *handlers) token(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "the body is not a form")
		return
	}
	var resp *TokenResponse
	var err error
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		resp, err = h.s.ExchangeCode(r.Context(), r.PostForm)
	case "refresh_token":
		resp, err = h.s.Refresh(r.Context(), r.PostForm)
	default:
		err = ErrUnsupportedGrantType
	}
	if err != nil {
		status := statusFor(err)
		if status >= 500 {
			h.logger.Error("token request failed", "error", err)
		}
		writeOAuthError(w, status, oauthCode(err), "")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) revoke(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err == nil {
		if err := h.s.Revoke(r.Context(), r.PostForm.Get("token")); err != nil {
			h.logger.Error("token revocation failed", "error", err)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// authorize validates the request and hands the browser to the app's consent
// page, where the person signs in if needed and decides.
func (h *handlers) authorize(w http.ResponseWriter, r *http.Request) {
	_, err := h.s.Prepare(r.Context(), r.URL.Query())
	var redirectErr *RedirectError
	switch {
	case errors.As(err, &redirectErr):
		http.Redirect(w, r, h.s.redirect(redirectErr.RedirectURI, url.Values{
			"error": {oauthCode(err)}, "error_description": {err.Error()}, "state": {redirectErr.State},
		}), http.StatusFound)
	case err != nil:
		if statusFor(err) >= 500 {
			h.logger.Error("authorization request failed", "error", err)
		}
		// The client or redirect could not be trusted, so the person is told
		// instead of being sent anywhere.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = errorPage.Execute(w, err.Error())
	default:
		http.Redirect(w, r, h.s.ConsentURL(r.URL.Query()), http.StatusFound)
	}
}

var errorPage = template.Must(template.New("error").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Sign in link not valid - Plonkout</title>
<style>body{font-family:system-ui,sans-serif;max-width:32rem;margin:3rem auto;padding:0 1rem;line-height:1.5}
code{background:#eee;padding:.1rem .3rem;border-radius:.2rem}</style></head>
<body><h1>This sign in link is not valid</h1>
<p>The app that sent you here asked for something Plonkout cannot allow, so nothing was shared.</p>
<p><code>{{.}}</code></p></body></html>`))

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	body := map[string]string{"error": code}
	if description != "" {
		body["error_description"] = description
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, body)
}

// AuthorizationParams are the /authorize parameters, carried through the
// consent page.
type AuthorizationParams struct {
	ClientID            string `json:"client_id" query:"client_id"`
	RedirectURI         string `json:"redirect_uri" query:"redirect_uri"`
	ResponseType        string `json:"response_type" query:"response_type"`
	CodeChallenge       string `json:"code_challenge" query:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method" query:"code_challenge_method"`
	Scope               string `json:"scope,omitempty" query:"scope" required:"false"`
	State               string `json:"state,omitempty" query:"state" required:"false"`
	Resource            string `json:"resource,omitempty" query:"resource" required:"false"`
}

func (p AuthorizationParams) values() url.Values {
	values := url.Values{}
	for key, value := range map[string]string{
		"client_id": p.ClientID, "redirect_uri": p.RedirectURI, "response_type": p.ResponseType,
		"code_challenge": p.CodeChallenge, "code_challenge_method": p.CodeChallengeMethod,
		"scope": p.Scope, "state": p.State, "resource": p.Resource,
	} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}

// ConsentPrompt is what the consent page shows.
type ConsentPrompt struct {
	ClientName   string   `json:"clientName"`
	ClientID     string   `json:"clientId"`
	RedirectHost string   `json:"redirectHost" doc:"Where the answer goes. Shown so a person can spot an impostor."`
	Loopback     bool     `json:"loopback" doc:"The answer goes to a program on this computer, e.g. Claude Code."`
	Scopes       []string `json:"scopes"`
}

// RegisterAPI declares the consent and connected apps operations.
func RegisterAPI(reg *api.Registry, s *Service) {
	tags := []string{"oauth"}

	problem := func(err error) error {
		if statusFor(err) >= 500 {
			return err
		}
		return api.Errorf(http.StatusUnprocessableEntity, oauthCode(err), "%s", err.Error())
	}

	api.Register(reg, api.Op{
		ID: "get-consent-prompt", Method: http.MethodGet, Path: "/oauth/consent",
		Summary: "Describe an authorization request for the consent page", Tags: tags,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *AuthorizationParams) (*struct{ Body ConsentPrompt }, error) {
		req, err := s.Prepare(ctx, in.values())
		if err != nil {
			return nil, problem(err)
		}
		redirect, _ := url.Parse(req.RedirectURI)
		prompt := ConsentPrompt{
			ClientName: req.ClientName, ClientID: req.ClientID, Scopes: []string{},
			RedirectHost: redirect.Host, Loopback: redirect.Scheme == "http" && isLoopback(redirect.Hostname()),
		}
		prompt.Scopes = append(prompt.Scopes, strings.Fields(req.Scope)...)
		return &struct{ Body ConsentPrompt }{Body: prompt}, nil
	})

	type decision struct {
		Params  AuthorizationParams `json:"params"`
		Approve bool                `json:"approve"`
	}
	api.Register(reg, api.Op{
		ID: "decide-consent", Method: http.MethodPost, Path: "/oauth/consent",
		Summary:     "Approve or deny an authorization request",
		Description: "Answers with where to send the browser: back to the client with a code, or with access_denied.",
		Tags:        tags, Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body decision }) (*struct {
		Body struct {
			RedirectTo string `json:"redirectTo"`
		}
	}, error) {
		target, err := s.Decide(ctx, api.UserID(ctx), in.Body.Params.values(), in.Body.Approve)
		if err != nil {
			return nil, problem(err)
		}
		out := &struct {
			Body struct {
				RedirectTo string `json:"redirectTo"`
			}
		}{}
		out.Body.RedirectTo = target
		return out, nil
	})

	api.Register(reg, api.Op{
		ID: "list-connected-apps", Method: http.MethodGet, Path: "/account/connected-apps",
		Summary: "Apps the user has let into their account, e.g. Claude", Tags: tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []ConnectedApp }, error) {
		apps, err := s.ConnectedApps(ctx, api.UserID(ctx))
		if err != nil {
			return nil, err
		}
		return &struct{ Body []ConnectedApp }{Body: apps}, nil
	})

	api.Register(reg, api.Op{
		ID: "disconnect-app", Method: http.MethodDelete, Path: "/account/connected-apps/{id}",
		Summary: "Revoke an app's access", Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.Disconnect(ctx, api.UserID(ctx), in.ID)
	})
}
