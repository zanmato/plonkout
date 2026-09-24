package oauth_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zanmato/plonkout/server/internal/platform/apitest"
)

const verifier = "a-very-long-code-verifier-that-is-at-least-forty-three-characters"

func challenge() string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// raw sends a request to the non API endpoints, the way an OAuth client does.
func raw(h *apitest.Harness, method, path string, body string, contentType string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, values := range header {
		req.Header[key] = values
	}
	rec := httptest.NewRecorder()
	h.Handler.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, into any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
}

func register(t *testing.T, h *apitest.Harness, redirects ...string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"client_name": "Claude", "redirect_uris": redirects, "token_endpoint_auth_method": "none",
		"grant_types": []string{"authorization_code", "refresh_token"},
	})
	rec := raw(h, http.MethodPost, "/oauth/register", string(body), "application/json", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	decode(t, rec, &out)
	return out.ClientID
}

func authorizeParams(clientID, redirect string) url.Values {
	return url.Values{
		"client_id": {clientID}, "redirect_uri": {redirect}, "response_type": {"code"},
		"code_challenge": {challenge()}, "code_challenge_method": {"S256"}, "state": {"xyz"},
		"scope": {"training offline_access"}, "resource": {apitest.Origin + "/mcp"},
	}
}

func paramsBody(params url.Values) map[string]string {
	out := map[string]string{}
	for key := range params {
		out[key] = params.Get(key)
	}
	return out
}

// signIn walks the whole authorization code flow and returns the tokens.
func signIn(t *testing.T, h *apitest.Harness, user apitest.User, clientID, redirect string) (access, refresh string) {
	t.Helper()
	params := authorizeParams(clientID, redirect)

	rec := raw(h, http.MethodGet, "/oauth/authorize?"+params.Encode(), "", "", nil)
	if rec.Code != http.StatusFound || !strings.HasPrefix(rec.Header().Get("Location"), apitest.Origin+"/#/authorize?") {
		t.Fatalf("authorize should hand over to the consent page, got %d %s", rec.Code, rec.Header().Get("Location"))
	}

	var prompt struct {
		ClientName   string   `json:"clientName"`
		RedirectHost string   `json:"redirectHost"`
		Scopes       []string `json:"scopes"`
	}
	h.Expect(h.Do(http.MethodGet, "/oauth/consent?"+params.Encode(), nil, user.Session), http.StatusOK).Decode(t, &prompt)
	if prompt.ClientName != "Claude" || len(prompt.Scopes) != 2 {
		t.Fatalf("unexpected consent prompt %+v", prompt)
	}

	var decided struct {
		RedirectTo string `json:"redirectTo"`
	}
	h.Expect(h.Do(http.MethodPost, "/oauth/consent", map[string]any{"params": paramsBody(params), "approve": true}, user.Session), http.StatusOK).Decode(t, &decided)
	target, err := url.Parse(decided.RedirectTo)
	if err != nil {
		t.Fatal(err)
	}
	q := target.Query()
	if q.Get("state") != "xyz" || q.Get("iss") != apitest.Origin || q.Get("code") == "" {
		t.Fatalf("unexpected redirect %s", decided.RedirectTo)
	}

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {q.Get("code")}, "code_verifier": {verifier},
		"client_id": {clientID}, "redirect_uri": {redirect}, "resource": {apitest.Origin + "/mcp"},
	}
	tokens := raw(h, http.MethodPost, "/oauth/token", form.Encode(), "application/x-www-form-urlencoded", nil)
	if tokens.Code != http.StatusOK {
		t.Fatalf("token: %d %s", tokens.Code, tokens.Body.String())
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	decode(t, tokens, &out)
	if out.TokenType != "Bearer" || out.Scope != "training offline_access" || out.RefreshToken == "" {
		t.Fatalf("unexpected tokens %+v", out)
	}
	return out.AccessToken, out.RefreshToken
}

func bearerStatus(h *apitest.Harness, token string) int {
	return raw(h, http.MethodGet, "/api/account", "", "", http.Header{"Authorization": {"Bearer " + token}}).Code
}

func refresh(h *apitest.Harness, token string) *httptest.ResponseRecorder {
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}}
	return raw(h, http.MethodPost, "/oauth/token", form.Encode(), "application/x-www-form-urlencoded", nil)
}

func TestDiscovery(t *testing.T) {
	h := apitest.New(t)

	var server map[string]any
	decode(t, raw(h, http.MethodGet, "/.well-known/oauth-authorization-server", "", "", nil), &server)
	if server["issuer"] != apitest.Origin || server["client_id_metadata_document_supported"] != true ||
		server["registration_endpoint"] != apitest.Origin+"/oauth/register" {
		t.Fatalf("unexpected server metadata %v", server)
	}

	var resource map[string]any
	decode(t, raw(h, http.MethodGet, "/.well-known/oauth-protected-resource/mcp", "", "", nil), &resource)
	if resource["resource"] != apitest.Origin+"/mcp" {
		t.Fatalf("unexpected resource metadata %v", resource)
	}
}

func TestAuthorizationCodeFlowAndRefresh(t *testing.T) {
	h := apitest.New(t)
	user := h.NewUser()
	clientID := register(t, h, "https://claude.ai/api/mcp/auth_callback")
	access, refreshToken := signIn(t, h, user, clientID, "https://claude.ai/api/mcp/auth_callback")

	if status := bearerStatus(h, access); status != http.StatusOK {
		t.Fatalf("the access token should work on the API, got %d", status)
	}
	if status := bearerStatus(h, "not-a-token"); status != http.StatusUnauthorized {
		t.Fatalf("an unknown token must be refused, got %d", status)
	}

	rotated := refresh(h, refreshToken)
	if rotated.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rotated.Code, rotated.Body.String())
	}
	var next struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decode(t, rotated, &next)
	if next.RefreshToken == refreshToken {
		t.Fatal("the refresh token must rotate")
	}

	// The old refresh token coming back means somebody kept a copy. The grant
	// ends, so neither copy nor the new access token works any more.
	reused := refresh(h, refreshToken)
	var failure struct{ Error string }
	decode(t, reused, &failure)
	if reused.Code != http.StatusBadRequest || failure.Error != "invalid_grant" {
		t.Fatalf("reuse must be invalid_grant, got %d %s", reused.Code, reused.Body.String())
	}
	if refresh(h, next.RefreshToken).Code != http.StatusBadRequest {
		t.Fatal("the grant should be revoked after a reuse")
	}
	if bearerStatus(h, next.AccessToken) != http.StatusUnauthorized {
		t.Fatal("access tokens of a revoked grant must stop working")
	}
}

func TestCodeIsBoundToVerifierAndUsedOnce(t *testing.T) {
	h := apitest.New(t)
	user := h.NewUser()
	redirect := "https://claude.ai/api/mcp/auth_callback"
	clientID := register(t, h, redirect)
	params := authorizeParams(clientID, redirect)

	code := func() string {
		var decided struct {
			RedirectTo string `json:"redirectTo"`
		}
		h.Expect(h.Do(http.MethodPost, "/oauth/consent", map[string]any{"params": paramsBody(params), "approve": true}, user.Session), http.StatusOK).Decode(t, &decided)
		target, _ := url.Parse(decided.RedirectTo)
		return target.Query().Get("code")
	}
	exchange := func(code, verifier string) int {
		form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "client_id": {clientID}, "redirect_uri": {redirect}}
		return raw(h, http.MethodPost, "/oauth/token", form.Encode(), "application/x-www-form-urlencoded", nil).Code
	}

	if exchange(code(), "the-wrong-verifier-the-wrong-verifier-the-wrong") != http.StatusBadRequest {
		t.Fatal("a wrong verifier must fail")
	}
	good := code()
	if exchange(good, verifier) != http.StatusOK {
		t.Fatal("the right verifier must work")
	}
	if exchange(good, verifier) != http.StatusBadRequest {
		t.Fatal("a code works once")
	}
}

func TestDeniedConsent(t *testing.T) {
	h := apitest.New(t)
	user := h.NewUser()
	redirect := "https://claude.ai/api/mcp/auth_callback"
	params := authorizeParams(register(t, h, redirect), redirect)

	var decided struct {
		RedirectTo string `json:"redirectTo"`
	}
	h.Expect(h.Do(http.MethodPost, "/oauth/consent", map[string]any{"params": paramsBody(params), "approve": false}, user.Session), http.StatusOK).Decode(t, &decided)
	if !strings.Contains(decided.RedirectTo, "error=access_denied") || !strings.Contains(decided.RedirectTo, "state=xyz") {
		t.Fatalf("unexpected denial redirect %s", decided.RedirectTo)
	}
}

func TestConsentNeedsASignedInUser(t *testing.T) {
	h := apitest.New(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	params := authorizeParams(register(t, h, redirect), redirect)
	h.Expect(h.Do(http.MethodPost, "/oauth/consent", map[string]any{"params": paramsBody(params), "approve": true}, ""), http.StatusUnauthorized)
}

func TestAuthorizeRefusals(t *testing.T) {
	h := apitest.New(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	clientID := register(t, h, redirect)

	t.Run("unregistered redirect shows a page instead of redirecting", func(t *testing.T) {
		params := authorizeParams(clientID, "https://evil.example/callback")
		rec := raw(h, http.MethodGet, "/oauth/authorize?"+params.Encode(), "", "", nil)
		if rec.Code != http.StatusBadRequest || rec.Header().Get("Location") != "" {
			t.Fatalf("expected an error page, got %d %s", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("wrong resource is reported to the client", func(t *testing.T) {
		params := authorizeParams(clientID, redirect)
		params.Set("resource", "https://other.example/mcp")
		rec := raw(h, http.MethodGet, "/oauth/authorize?"+params.Encode(), "", "", nil)
		if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "error=invalid_request") {
			t.Fatalf("expected an error redirect, got %d %s", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("no PKCE", func(t *testing.T) {
		params := authorizeParams(clientID, redirect)
		params.Del("code_challenge")
		rec := raw(h, http.MethodGet, "/oauth/authorize?"+params.Encode(), "", "", nil)
		if !strings.Contains(rec.Header().Get("Location"), "error=invalid_request") {
			t.Fatalf("expected an error redirect, got %d %s", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("http to a public host cannot register", func(t *testing.T) {
		body := `{"redirect_uris":["http://evil.example/callback"]}`
		rec := raw(h, http.MethodPost, "/oauth/register", body, "application/json", nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected refusal, got %d", rec.Code)
		}
	})
}

func TestLoopbackRedirectIgnoresPort(t *testing.T) {
	h := apitest.New(t)
	user := h.NewUser()
	clientID := register(t, h, "http://127.0.0.1/callback")

	// Claude Code listens on a free port each time it signs in.
	access, _ := signIn(t, h, user, clientID, "http://127.0.0.1:54321/callback")
	if bearerStatus(h, access) != http.StatusOK {
		t.Fatal("the token from a loopback sign in should work")
	}

	params := authorizeParams(clientID, "http://127.0.0.1:54321/other")
	if rec := raw(h, http.MethodGet, "/oauth/authorize?"+params.Encode(), "", "", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("only the port may differ, got %d", rec.Code)
	}
}

func TestConnectedApps(t *testing.T) {
	h := apitest.New(t)
	user, other := h.NewUser(), h.NewUser()
	redirect := "https://claude.ai/api/mcp/auth_callback"
	access, _ := signIn(t, h, user, register(t, h, redirect), redirect)

	var apps []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	h.Expect(h.Do(http.MethodGet, "/account/connected-apps", nil, user.Session), http.StatusOK).Decode(t, &apps)
	if len(apps) != 1 || apps[0].Name != "Claude" {
		t.Fatalf("unexpected apps %+v", apps)
	}
	h.Expect(h.Do(http.MethodDelete, "/account/connected-apps/"+apps[0].ID, nil, other.Session), http.StatusNotFound)
	h.Expect(h.Do(http.MethodDelete, "/account/connected-apps/"+apps[0].ID, nil, user.Session), http.StatusNoContent)
	if bearerStatus(h, access) != http.StatusUnauthorized {
		t.Fatal("disconnecting an app revokes its tokens")
	}
}

func TestBearerTokensSkipTheOriginCheck(t *testing.T) {
	h := apitest.New(t)
	user := h.NewUser()
	redirect := "https://claude.ai/api/mcp/auth_callback"
	access, _ := signIn(t, h, user, register(t, h, redirect), redirect)

	// Tokens do not ride along on requests by themselves like cookies do, so
	// the CSRF check does not apply.
	rec := raw(h, http.MethodPut, "/api/settings/weightUnit", `{"value":"kg"}`, "application/json", http.Header{
		"Authorization": {"Bearer " + access}, "Origin": {"https://claude.ai"},
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected a write with a token to work, got %d %s", rec.Code, rec.Body.String())
	}
}
