package account_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
)

var rp = virtualwebauthn.RelyingParty{ID: "localhost", Name: "Plonkout", Origin: apitest.Origin}

// harness adds the passkey flows to the shared API harness.
type harness struct {
	*apitest.Harness
	t *testing.T
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return &harness{Harness: apitest.New(t), t: t}
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
	h.Expect(h.Do(http.MethodGet, "/auth/signup/challenge", nil, ""), http.StatusOK).Decode(h.t, &c)
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
	h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{
		"username": username, "pow": h.solve(),
	}, ""), http.StatusOK).Decode(h.t, &begin)

	phone := newDevice()
	finished := h.Expect(h.Do(http.MethodPost, "/auth/signup/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": phone.create(h.t, begin), "passkeyName": "Phone",
	}, ""), http.StatusOK)

	var out signedIn
	finished.Decode(h.t, &out)
	if finished.Cookie == "" {
		h.t.Fatal("signup did not set a session cookie")
	}
	return phone, finished.Cookie, out
}

func (h *harness) login(d *device) apitest.Response {
	h.t.Helper()
	var begin ceremony
	h.Expect(h.Do(http.MethodPost, "/auth/login/begin", nil, ""), http.StatusOK).Decode(h.t, &begin)
	return h.Do(http.MethodPost, "/auth/login/finish", map[string]any{
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
	h.Expect(h.Do(http.MethodGet, "/account", nil, session), http.StatusOK).Decode(t, &who)
	if who.Username != "andreas" || who.Passkeys != 1 || who.RecoveryCodesLeft != 10 {
		t.Fatalf("unexpected account %+v", who)
	}

	// The default exercises were written as the new user, through RLS.
	var exercises int
	if err := h.DB.Owner.QueryRow(t.Context(), `SELECT count(*) FROM exercises WHERE user_id = $1`, out.User.ID).Scan(&exercises); err != nil {
		t.Fatal(err)
	}
	if exercises != 67 {
		t.Fatalf("expected 67 seeded exercises, got %d", exercises)
	}

	h.Expect(h.Do(http.MethodPost, "/auth/logout", nil, session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodGet, "/account", nil, session), http.StatusUnauthorized)

	again := h.Expect(h.login(phone), http.StatusOK)
	h.Expect(h.Do(http.MethodGet, "/account", nil, again.Cookie), http.StatusOK)
}

func TestSignupRefusals(t *testing.T) {
	h := newHarness(t)
	h.signup("taken")

	t.Run("taken username", func(t *testing.T) {
		r := h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "Taken", "pow": h.solve()}, ""), http.StatusConflict)
		if r.Code(t) != "username_taken" {
			t.Fatalf("unexpected problem %s", r.Body)
		}
	})

	t.Run("honeypot", func(t *testing.T) {
		h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{
			"username": "bot", "website": "http://spam", "pow": h.solve(),
		}, ""), http.StatusUnprocessableEntity)
	})

	t.Run("reused proof of work", func(t *testing.T) {
		solution := h.solve()
		h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "first", "pow": solution}, ""), http.StatusOK)
		r := h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "second", "pow": solution}, ""), http.StatusUnprocessableEntity)
		if r.Code(t) != "signup_refused" {
			t.Fatalf("unexpected problem %s", r.Body)
		}
	})

	t.Run("wrong number", func(t *testing.T) {
		solution := h.solve()
		solution["number"] = solution["number"].(int) + 1
		h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "third", "pow": solution}, ""), http.StatusUnprocessableEntity)
	})

	t.Run("forged signature", func(t *testing.T) {
		solution := h.solve()
		solution["signature"] = strings.Repeat("0", 64)
		h.Expect(h.Do(http.MethodPost, "/auth/signup/begin", map[string]any{"username": "fourth", "pow": solution}, ""), http.StatusUnprocessableEntity)
	})
}

func TestCeremonyCannotBeFinishedTwice(t *testing.T) {
	h := newHarness(t)
	phone, _, _ := h.signup("once")

	var begin ceremony
	h.Expect(h.Do(http.MethodPost, "/auth/login/begin", nil, ""), http.StatusOK).Decode(t, &begin)
	body := map[string]any{"ceremony": begin.Ceremony, "credential": phone.get(t, begin)}
	h.Expect(h.Do(http.MethodPost, "/auth/login/finish", body, ""), http.StatusOK)
	r := h.Expect(h.Do(http.MethodPost, "/auth/login/finish", body, ""), http.StatusUnprocessableEntity)
	if r.Code(t) != "ceremony_expired" {
		t.Fatalf("unexpected problem %s", r.Body)
	}
}

func TestUnknownPasskeyCannotSignIn(t *testing.T) {
	h := newHarness(t)
	h.signup("real")

	stranger := newDevice()
	stranger.authenticator.Options.UserHandle = []byte("not a user of this site")
	stranger.authenticator.AddCredential(stranger.credential)
	h.Expect(h.login(stranger), http.StatusUnauthorized)
}

func TestRecovery(t *testing.T) {
	h := newHarness(t)
	_, _, out := h.signup("forgetful")
	code := out.RecoveryCodes[3]

	h.Expect(h.Do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "forgetful", "recoveryCode": "wrong-code0"}, ""), http.StatusUnauthorized)
	h.Expect(h.Do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "nobody", "recoveryCode": code}, ""), http.StatusUnauthorized)

	// Codes are forgiving about case and dashes, the way people type them.
	var begin ceremony
	typed := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	h.Expect(h.Do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "forgetful", "recoveryCode": typed}, ""), http.StatusOK).Decode(t, &begin)

	newPhone := newDevice()
	recovered := h.Expect(h.Do(http.MethodPost, "/auth/recover/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": newPhone.create(t, begin), "passkeyName": "New phone",
	}, ""), http.StatusOK)

	var who me
	h.Expect(h.Do(http.MethodGet, "/account", nil, recovered.Cookie), http.StatusOK).Decode(t, &who)
	if who.Passkeys != 2 || who.RecoveryCodesLeft != 9 {
		t.Fatalf("unexpected account after recovery %+v", who)
	}

	h.Expect(h.Do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "forgetful", "recoveryCode": code}, ""), http.StatusUnauthorized)
	h.Expect(h.login(newPhone), http.StatusOK)
}

func TestPasskeyManagement(t *testing.T) {
	h := newHarness(t)
	_, session, _ := h.signup("collector")

	var begin ceremony
	h.Expect(h.Do(http.MethodPost, "/account/passkeys/begin", nil, session), http.StatusOK).Decode(t, &begin)
	laptop := newDevice()
	h.Expect(h.Do(http.MethodPost, "/account/passkeys/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": laptop.create(t, begin), "passkeyName": "Laptop",
	}, session), http.StatusCreated)

	type passkey struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Synced bool   `json:"synced"`
	}
	var passkeys []passkey
	h.Expect(h.Do(http.MethodGet, "/account/passkeys", nil, session), http.StatusOK).Decode(t, &passkeys)
	if len(passkeys) != 2 || passkeys[0].Name != "Phone" || passkeys[1].Name != "Laptop" || !passkeys[0].Synced {
		t.Fatalf("unexpected passkeys %+v", passkeys)
	}

	h.Expect(h.Do(http.MethodPatch, "/account/passkeys/"+passkeys[1].ID, map[string]any{"name": "Work laptop"}, session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodDelete, "/account/passkeys/"+passkeys[0].ID, nil, session), http.StatusNoContent)
	r := h.Expect(h.Do(http.MethodDelete, "/account/passkeys/"+passkeys[1].ID, nil, session), http.StatusConflict)
	if r.Code(t) != "last_passkey" {
		t.Fatalf("unexpected problem %s", r.Body)
	}

	// The remaining passkey is the laptop, and it still signs in.
	h.Expect(h.login(laptop), http.StatusOK)
}

func TestPasskeysAreNamedByProvider(t *testing.T) {
	h := newHarness(t)
	_, session, _ := h.signup("named")

	var begin ceremony
	h.Expect(h.Do(http.MethodPost, "/account/passkeys/begin", nil, session), http.StatusOK).Decode(t, &begin)
	// A passkey created on a Linux laptop by scanning the QR code with an
	// Android phone lives in Google Password Manager, whatever the laptop says.
	phone := newDevice()
	aaguid, _ := hex.DecodeString("ea9b8d664d011d213ce4b6b48cb575d4")
	copy(phone.authenticator.Aaguid[:], aaguid)
	h.Expect(h.Do(http.MethodPost, "/account/passkeys/finish", map[string]any{
		"ceremony": begin.Ceremony, "credential": phone.create(t, begin), "passkeyName": "Linux",
	}, session), http.StatusCreated)

	var passkeys []struct{ Name string }
	h.Expect(h.Do(http.MethodGet, "/account/passkeys", nil, session), http.StatusOK).Decode(t, &passkeys)
	// The first came from an unknown authenticator, so the browser's guess stays.
	if len(passkeys) != 2 || passkeys[0].Name != "Phone" || passkeys[1].Name != "Google Password Manager" {
		t.Fatalf("unexpected names %+v", passkeys)
	}
}

func TestPasskeysBelongToTheirUser(t *testing.T) {
	h := newHarness(t)
	_, alice, _ := h.signup("alice")
	_, bob, _ := h.signup("bob")

	var passkeys []struct {
		ID string `json:"id"`
	}
	h.Expect(h.Do(http.MethodGet, "/account/passkeys", nil, alice), http.StatusOK).Decode(t, &passkeys)
	h.Expect(h.Do(http.MethodPatch, "/account/passkeys/"+passkeys[0].ID, map[string]any{"name": "mine now"}, bob), http.StatusNotFound)
}

func TestRegenerateRecoveryCodes(t *testing.T) {
	h := newHarness(t)
	_, session, out := h.signup("careful")

	var codes []string
	h.Expect(h.Do(http.MethodPost, "/account/recovery-codes", nil, session), http.StatusOK).Decode(t, &codes)
	if len(codes) != 10 || codes[0] == out.RecoveryCodes[0] {
		t.Fatalf("expected a fresh set of codes, got %v", codes)
	}
	h.Expect(h.Do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "careful", "recoveryCode": out.RecoveryCodes[0]}, ""), http.StatusUnauthorized)
	h.Expect(h.Do(http.MethodPost, "/auth/recover/begin", map[string]any{"username": "careful", "recoveryCode": codes[0]}, ""), http.StatusOK)
}

func TestCookieWritesMustComeFromTheApp(t *testing.T) {
	h := newHarness(t)
	_, session, _ := h.signup("targeted")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: api.SessionCookie, Value: session})
	rec := httptest.NewRecorder()
	h.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected a cross origin write to be refused, got %d", rec.Code)
	}
	h.Expect(h.Do(http.MethodGet, "/account", nil, session), http.StatusOK)
}

func TestDeleteAccount(t *testing.T) {
	h := newHarness(t)
	phone, session, out := h.signup("leaving")

	h.Expect(h.Do(http.MethodDelete, "/account", nil, session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodGet, "/account", nil, session), http.StatusUnauthorized)
	h.Expect(h.login(phone), http.StatusUnauthorized)

	var left int
	if err := h.DB.Owner.QueryRow(t.Context(), `SELECT count(*) FROM exercises WHERE user_id = $1`, out.User.ID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("a deleted user still owns %d exercises", left)
	}
}
