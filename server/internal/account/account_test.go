package account_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/dbtest"
	"github.com/zanmato/plonkout/server/internal/server"
)

const origin = "http://localhost:5173"

var rp = virtualwebauthn.RelyingParty{ID: "localhost", Name: "Plonkout", Origin: origin}

// harness drives the real handler against a real database.
type harness struct {
	t       *testing.T
	db      *dbtest.DB
	handler http.Handler
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	d := dbtest.New(t)
	cfg := &config.Config{
		Server:   config.Server{BaseURL: origin},
		Auth:     config.Auth{SecretKey: "test-secret-key-that-is-long-enough", SessionTTL: config.Duration(3600e9), SessionAbsoluteTTL: config.Duration(7200e9)},
		WebAuthn: config.WebAuthn{RPID: "localhost", RPName: "Plonkout", Origins: []string{origin}},
	}
	srv, err := server.New(server.Deps{Config: cfg, Pool: d.App, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, db: d, handler: srv.Handler()}
}

type response struct {
	status int
	body   []byte
	cookie string
}

func (r response) decode(t *testing.T, into any) {
	t.Helper()
	if err := json.Unmarshal(r.body, into); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (r response) code(t *testing.T) string {
	t.Helper()
	var problem api.Problem
	r.decode(t, &problem)
	return problem.Code
}

// do sends a request as the SPA would, with the session cookie by hand since
// the __Host- cookie is Secure and httptest is plain http.
func (h *harness) do(method, path string, body any, session string) response {
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
	req.Header.Set("Origin", origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		req.AddCookie(&http.Cookie{Name: api.SessionCookie, Value: session})
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)

	out := response{status: rec.Code, body: rec.Body.Bytes()}

	for _, c := range rec.Result().Cookies() {
		if c.Name == api.SessionCookie {
			out.cookie = c.Value
		}
	}
	return out
}

func (h *harness) expect(r response, status int) response {
	h.t.Helper()
	if r.status != status {
		h.t.Fatalf("expected %d, got %d: %s", status, r.status, r.body)
	}
	return r
}

type ceremony struct {
	Ceremony string          `json:"ceremony"`
	Options  json.RawMessage `json:"options"`
}

// device is a phone with a passkey on it.
type device struct {
	authenticator virtualwebauthn.Authenticator
	credential    virtualwebauthn.Credential
}

func newDevice() *device {
	return &device{
		authenticator: virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{BackupEligible: true, BackupState: true}),
		credential:    virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2),
	}
}

// create answers a registration ceremony like navigator.credentials.create.
func (d *device) create(t *testing.T, c ceremony) json.RawMessage {
	t.Helper()
	options, err := virtualwebauthn.ParseAttestationOptions(string(c.Options))
	if err != nil {
		t.Fatalf("parse attestation options: %v", err)
	}
	// A discoverable passkey remembers whose it is, so login needs no username.
	// The parser has already decoded the user handle.
	d.authenticator.Options.UserHandle = []byte(options.UserID)
	d.authenticator.AddCredential(d.credential)
	return json.RawMessage(virtualwebauthn.CreateAttestationResponse(rp, d.authenticator, d.credential, *options))
}

// get answers a login ceremony like navigator.credentials.get.
func (d *device) get(t *testing.T, c ceremony) json.RawMessage {
	t.Helper()
	options, err := virtualwebauthn.ParseAssertionOptions(string(c.Options))
	if err != nil {
		t.Fatalf("parse assertion options: %v", err)
	}
	return json.RawMessage(virtualwebauthn.CreateAssertionResponse(rp, d.authenticator, d.credential, *options))
}

type powChallenge struct {
	Challenge string `json:"challenge"`
	MaxNumber int    `json:"maxNumber"`
	Salt      string `json:"salt"`
	Signature string `json:"signature"`
}

// solve does what the browser's worker does.
func (h *harness) solve() map[string]any {
	h.t.Helper()
	var c powChallenge
	h.expect(h.do(http.MethodGet, "/auth/signup/challenge", nil, ""), http.StatusOK).decode(h.t, &c)
	for n := 0; n <= c.MaxNumber; n++ {
		sum := sha256.Sum256([]byte(c.Salt + strconv.Itoa(n)))
		if hex.EncodeToString(sum[:]) == c.Challenge {
			return map[string]any{"challenge": c.Challenge, "number": n, "salt": c.Salt, "signature": c.Signature}
		}
	}
	h.t.Fatal("the proof of work has no solution")
	return nil
}

type signedIn struct {
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
	RecoveryCodes []string `json:"recoveryCodes"`
}

// signup creates an account and returns the device, the session and the
// recovery codes.
func (h *harness) signup(username string) (*device, string, signedIn) {
	h.t.Helper()
	var begin ceremony
	h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{
		"username": username, "pow": h.solve(),
	}, ""), http.StatusOK).decode(h.t, &begin)

	phone := newDevice()
	finished := h.expect(h.do(http.MethodPost, "/auth/signup/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": phone.create(h.t, begin), "passkeyName": "Phone",
	}, ""), http.StatusOK)

	var out signedIn
	finished.decode(h.t, &out)
	if finished.cookie == "" {
		h.t.Fatal("signup did not set a session cookie")
	}
	return phone, finished.cookie, out
}

func (h *harness) login(d *device) response {
	h.t.Helper()
	var begin ceremony
	h.expect(h.do(http.MethodPost, "/auth/login/begin", nil, ""), http.StatusOK).decode(h.t, &begin)
	return h.do(http.MethodPost, "/auth/login/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": d.get(h.t, begin),
	}, "")
}

type me struct {
	Username          string `json:"username"`
	Passkeys          int    `json:"passkeys"`
	RecoveryCodesLeft int    `json:"recoveryCodesLeft"`
}

func TestSignupLoginLogout(t *testing.T) {
	h := newHarness(t)
	phone, session, out := h.signup("andreas")

	if len(out.RecoveryCodes) != 10 {
		t.Fatalf("expected 10 recovery codes, got %d", len(out.RecoveryCodes))
	}

	var who me
	h.expect(h.do(http.MethodGet, "/account", nil, session), http.StatusOK).decode(t, &who)
	if who.Username != "andreas" || who.Passkeys != 1 || who.RecoveryCodesLeft != 10 {
		t.Fatalf("unexpected account %+v", who)
	}

	// The default exercises were written as the new user, through RLS.
	var exercises int
	if err := h.db.Owner.QueryRow(t.Context(), `SELECT count(*) FROM exercises WHERE user_id = $1`, out.User.ID).Scan(&exercises); err != nil {
		t.Fatal(err)
	}
	if exercises != 67 {
		t.Fatalf("expected 67 seeded exercises, got %d", exercises)
	}

	h.expect(h.do(http.MethodPost, "/auth/logout", nil, session), http.StatusNoContent)
	h.expect(h.do(http.MethodGet, "/account", nil, session), http.StatusUnauthorized)

	again := h.expect(h.login(phone), http.StatusOK)
	h.expect(h.do(http.MethodGet, "/account", nil, again.cookie), http.StatusOK)
}

func TestSignupRefusals(t *testing.T) {
	h := newHarness(t)
	h.signup("taken")

	t.Run("taken username", func(t *testing.T) {
		r := h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "Taken", "pow": h.solve()}, ""), http.StatusConflict)
		if r.code(t) != "username_taken" {
			t.Fatalf("unexpected problem %s", r.body)
		}
	})

	t.Run("honeypot", func(t *testing.T) {
		h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{
			"username": "bot", "website": "http://spam", "pow": h.solve(),
		}, ""), http.StatusUnprocessableEntity)
	})

	t.Run("reused proof of work", func(t *testing.T) {
		solution := h.solve()
		h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "first", "pow": solution}, ""), http.StatusOK)
		r := h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "second", "pow": solution}, ""), http.StatusUnprocessableEntity)
		if r.code(t) != "signup_refused" {
			t.Fatalf("unexpected problem %s", r.body)
		}
	})

	t.Run("wrong number", func(t *testing.T) {
		solution := h.solve()
		solution["number"] = solution["number"].(int) + 1
		h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "third", "pow": solution}, ""), http.StatusUnprocessableEntity)
	})

	t.Run("forged signature", func(t *testing.T) {
		solution := h.solve()
		solution["signature"] = strings.Repeat("0", 64)
		h.expect(h.do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "fourth", "pow": solution}, ""), http.StatusUnprocessableEntity)
	})
}

func TestCeremonyCannotBeFinishedTwice(t *testing.T) {
	h := newHarness(t)
	phone, _, _ := h.signup("once")

	var begin ceremony
	h.expect(h.do(http.MethodPost, "/auth/login/begin", nil, ""), http.StatusOK).decode(t, &begin)
	body := map[string]any{"ceremony": begin.Ceremony, "credential": phone.get(t, begin)}
	h.expect(h.do(http.MethodPost, "/auth/login/finish", body, ""), http.StatusOK)
	r := h.expect(h.do(http.MethodPost, "/auth/login/finish", body, ""), http.StatusUnprocessableEntity)
	if r.code(t) != "ceremony_expired" {
		t.Fatalf("unexpected problem %s", r.body)
	}
}

func TestUnknownPasskeyCannotSignIn(t *testing.T) {
	h := newHarness(t)
	h.signup("real")

	stranger := newDevice()
	stranger.authenticator.Options.UserHandle = []byte("not a user of this site")
	stranger.authenticator.AddCredential(stranger.credential)
	h.expect(h.login(stranger), http.StatusUnauthorized)
}

func TestRecovery(t *testing.T) {
	h := newHarness(t)
	_, _, out := h.signup("forgetful")
	code := out.RecoveryCodes[3]

	h.expect(h.do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "forgetful", "recoveryCode": "wrong-code0"}, ""), http.StatusUnauthorized)
	h.expect(h.do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "nobody", "recoveryCode": code}, ""), http.StatusUnauthorized)

	// Codes are forgiving about case and dashes, the way people type them.
	var begin ceremony
	typed := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	h.expect(h.do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "forgetful", "recoveryCode": typed}, ""), http.StatusOK).decode(t, &begin)

	newPhone := newDevice()
	recovered := h.expect(h.do(http.MethodPost, "/auth/recover/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": newPhone.create(t, begin), "passkeyName": "New phone",
	}, ""), http.StatusOK)

	var who me
	h.expect(h.do(http.MethodGet, "/account", nil, recovered.cookie), http.StatusOK).decode(t, &who)
	if who.Passkeys != 2 || who.RecoveryCodesLeft != 9 {
		t.Fatalf("unexpected account after recovery %+v", who)
	}

	h.expect(h.do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "forgetful", "recoveryCode": code}, ""), http.StatusUnauthorized)
	h.expect(h.login(newPhone), http.StatusOK)
}

func TestPasskeyManagement(t *testing.T) {
	h := newHarness(t)
	_, session, _ := h.signup("collector")

	var begin ceremony
	h.expect(h.do(http.MethodPost, "/account/passkeys/begin", nil, session), http.StatusOK).decode(t, &begin)
	laptop := newDevice()
	h.expect(h.do(http.MethodPost, "/account/passkeys/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": laptop.create(t, begin), "passkeyName": "Laptop",
	}, session), http.StatusCreated)

	type passkey struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Synced bool   `json:"synced"`
	}
	var passkeys []passkey
	h.expect(h.do(http.MethodGet, "/account/passkeys", nil, session), http.StatusOK).decode(t, &passkeys)
	if len(passkeys) != 2 || passkeys[0].Name != "Phone" || passkeys[1].Name != "Laptop" || !passkeys[0].Synced {
		t.Fatalf("unexpected passkeys %+v", passkeys)
	}

	h.expect(h.do(http.MethodPatch, "/account/passkeys/"+passkeys[1].ID, map[string]any{"name": "Work laptop"}, session), http.StatusNoContent)
	h.expect(h.do(http.MethodDelete, "/account/passkeys/"+passkeys[0].ID, nil, session), http.StatusNoContent)
	r := h.expect(h.do(http.MethodDelete, "/account/passkeys/"+passkeys[1].ID, nil, session), http.StatusConflict)
	if r.code(t) != "last_passkey" {
		t.Fatalf("unexpected problem %s", r.body)
	}

	// The remaining passkey is the laptop, and it still signs in.
	h.expect(h.login(laptop), http.StatusOK)
}

func TestPasskeysBelongToTheirUser(t *testing.T) {
	h := newHarness(t)
	_, alice, _ := h.signup("alice")
	_, bob, _ := h.signup("bob")

	var passkeys []struct {
		ID string `json:"id"`
	}
	h.expect(h.do(http.MethodGet, "/account/passkeys", nil, alice), http.StatusOK).decode(t, &passkeys)
	h.expect(h.do(http.MethodPatch, "/account/passkeys/"+passkeys[0].ID, map[string]any{"name": "mine now"}, bob), http.StatusNotFound)
}

func TestRegenerateRecoveryCodes(t *testing.T) {
	h := newHarness(t)
	_, session, out := h.signup("careful")

	var codes []string
	h.expect(h.do(http.MethodPost, "/account/recovery-codes", nil, session), http.StatusOK).decode(t, &codes)
	if len(codes) != 10 || codes[0] == out.RecoveryCodes[0] {
		t.Fatalf("expected a fresh set of codes, got %v", codes)
	}
	h.expect(h.do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "careful", "recoveryCode": out.RecoveryCodes[0]}, ""), http.StatusUnauthorized)
	h.expect(h.do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "careful", "recoveryCode": codes[0]}, ""), http.StatusOK)
}

func TestCookieWritesMustComeFromTheApp(t *testing.T) {
	h := newHarness(t)
	_, session, _ := h.signup("targeted")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: api.SessionCookie, Value: session})
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected a cross origin write to be refused, got %d", rec.Code)
	}
	h.expect(h.do(http.MethodGet, "/account", nil, session), http.StatusOK)
}

func TestDeleteAccount(t *testing.T) {
	h := newHarness(t)
	phone, session, out := h.signup("leaving")

	h.expect(h.do(http.MethodDelete, "/account", nil, session), http.StatusNoContent)
	h.expect(h.do(http.MethodGet, "/account", nil, session), http.StatusUnauthorized)
	h.expect(h.login(phone), http.StatusUnauthorized)

	var left int
	if err := h.db.Owner.QueryRow(t.Context(), `SELECT count(*) FROM exercises WHERE user_id = $1`, out.User.ID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("a deleted user still owns %d exercises", left)
	}
}
