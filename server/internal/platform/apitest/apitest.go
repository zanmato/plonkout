// Package apitest drives the real HTTP handler against a real database.
package apitest

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/dbtest"
	"github.com/zanmato/plonkout/server/internal/server"
)

// Origin is the public origin the test server believes it runs at.
const Origin = "http://localhost:5173"

// Harness is one test's server and database.
type Harness struct {
	t       testing.TB
	DB      *dbtest.DB
	Handler http.Handler
}

// New builds the server on a fresh database.
func New(t testing.TB) *Harness {
	t.Helper()
	d := dbtest.New(t)
	cfg := &config.Config{
		Server: config.Server{BaseURL: Origin},
		Auth: config.Auth{
			SecretKey:          "test-secret-key-that-is-long-enough",
			SessionTTL:         config.Duration(time.Hour),
			SessionAbsoluteTTL: config.Duration(2 * time.Hour),
		},
		WebAuthn: config.WebAuthn{RPID: "localhost", RPName: "Plonkout", Origins: []string{Origin}},
		OAuth: config.OAuth{
			AccessTokenTTL:           config.Duration(time.Hour),
			RefreshTokenTTL:          config.Duration(24 * time.Hour),
			AllowDynamicRegistration: true,
		},
	}
	srv, err := server.New(server.Deps{Config: cfg, Pool: d.App, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return &Harness{t: t, DB: d, Handler: srv.Handler()}
}

// User is a signed in test user.
type User struct {
	ID      uuid.UUID
	Session string
}

// NewUser creates a user with a web session, without a passkey ceremony.
func (h *Harness) NewUser() User {
	h.t.Helper()
	id := h.DB.NewUser(h.t)

	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	_, err := h.DB.App.Exec(h.t.Context(), `
		INSERT INTO auth.web_sessions (token_hash, user_id, expires_at, absolute_expires_at)
		VALUES ($1, $2, now() + interval '1 hour', now() + interval '2 hours')`, hash[:], id)
	if err != nil {
		h.t.Fatalf("create a session: %v", err)
	}
	return User{ID: id, Session: token}
}

// Response is a recorded answer.
type Response struct {
	Status int
	Body   []byte
	Cookie string
}

// Decode reads the JSON body.
func (r Response) Decode(t testing.TB, into any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, into); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

// Code is the problem code of an error answer.
func (r Response) Code(t testing.TB) string {
	t.Helper()
	var problem api.Problem
	r.Decode(t, &problem)
	return problem.Code
}

// Do sends a request as the SPA would. The session cookie is added by hand,
// since the __Host- cookie is Secure and httptest is plain http.
func (h *Harness) Do(method, path string, body any, session string) Response {
	h.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			h.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, "/api"+path, reader)
	req.Header.Set("Origin", Origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		req.AddCookie(&http.Cookie{Name: api.SessionCookie, Value: session})
	}
	rec := httptest.NewRecorder()
	h.Handler.ServeHTTP(rec, req)

	out := Response{Status: rec.Code, Body: rec.Body.Bytes()}
	for _, c := range rec.Result().Cookies() {
		if c.Name == api.SessionCookie {
			out.Cookie = c.Value
		}
	}
	return out
}

// Expect fails the test unless the answer has the status.
func (h *Harness) Expect(r Response, status int) Response {
	h.t.Helper()
	if r.Status != status {
		h.t.Fatalf("expected %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

// NewAccessToken gives the user an OAuth access token for the MCP resource,
// as if an MCP client had been through the authorization flow.
func (h *Harness) NewAccessToken(u User) string {
	h.t.Helper()
	ctx := h.t.Context()
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))

	var grantID uuid.UUID
	err := h.DB.App.QueryRow(ctx, `
		WITH client AS (
			INSERT INTO auth.oauth_clients (client_id, kind, client_name, redirect_uris)
			VALUES ('test_' || gen_random_uuid(), 'dcr', 'Test client', '{https://client.example/callback}')
			RETURNING client_id
		)
		INSERT INTO auth.oauth_grants (client_id, user_id, scope, resource, refresh_expires_at)
		SELECT client_id, $1, 'training offline_access', $2, now() + interval '1 day' FROM client
		RETURNING id`, u.ID, Origin+"/mcp").Scan(&grantID)
	if err != nil {
		h.t.Fatalf("create a grant: %v", err)
	}
	if _, err := h.DB.App.Exec(ctx, `
		INSERT INTO auth.oauth_access_tokens (token_hash, grant_id, expires_at) VALUES ($1, $2, now() + interval '1 hour')`,
		hash[:], grantID); err != nil {
		h.t.Fatalf("create an access token: %v", err)
	}
	return token
}
