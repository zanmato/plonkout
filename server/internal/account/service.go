// Package account owns users, their passkeys, recovery codes and web sessions.
//
// Signing in is passkey only. A passkey with user verification is already two
// factors, the device and the biometric or PIN that unlocks it, and it cannot
// be phished because the browser binds it to this origin. Recovery codes are
// the way back when every passkey is gone.
package account

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/account/accountdb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/platform/userctx"
)

// defaultExercises is seeded for every new user. It started as the local only
// app's built in list and is edited here now.
//
//go:embed default_exercises.json
var defaultExercises []byte

const (
	ceremonyTTL = 5 * time.Minute
	// sessionTouchInterval limits how often a session's sliding expiry is
	// written, so a busy page is not a write per request.
	sessionTouchInterval = time.Hour
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)

// Ceremony kinds, matching the check constraint on auth.webauthn_ceremonies.
const (
	kindSignup     = "signup"
	kindLogin      = "login"
	kindAddPasskey = "add_passkey"
	kindRecover    = "recover"
)

var (
	errCeremony = api.Errorf(http.StatusUnprocessableEntity, "ceremony_expired",
		"the passkey prompt expired, try again")
	errPasskey = api.Errorf(http.StatusUnauthorized, "passkey_rejected",
		"the passkey could not be verified")
	errRecovery = api.Errorf(http.StatusUnauthorized, "recovery_rejected",
		"the username or recovery code is not right")
)

// Service is the account module.
type Service struct {
	pool     *pgxpool.Pool
	q        *accountdb.Queries
	webauthn *webauthn.WebAuthn
	pow      *pow

	sessionTTL         time.Duration
	sessionAbsoluteTTL time.Duration
	now                func() time.Time
}

// NewService builds the account module.
func NewService(pool *pgxpool.Pool, cfg *config.Config) (*Service, error) {
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.WebAuthn.RPID,
		RPDisplayName: cfg.WebAuthn.RPName,
		RPOrigins:     cfg.WebAuthn.Origins,
	})
	if err != nil {
		return nil, fmt.Errorf("configure passkeys: %w", err)
	}

	q := accountdb.New(pool)
	return &Service{
		pool:               pool,
		q:                  q,
		webauthn:           wa,
		pow:                &pow{key: deriveKey(cfg.Auth.SecretKey, "pow"), q: q, now: time.Now},
		sessionTTL:         cfg.Auth.SessionTTL.Duration(),
		sessionAbsoluteTTL: cfg.Auth.SessionAbsoluteTTL.Duration(),
		now:                time.Now,
	}, nil
}

// User is a signed in user as the API shows it.
type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Created  time.Time `json:"created"`
}

// Session is a new web session.
type Session struct {
	Token   string
	Expires time.Time
}

// ceremonyState is what a ceremony keeps between begin and finish.
type ceremonyState struct {
	WebAuthn webauthn.SessionData `json:"webauthn"`
	// RecoveryCodeHash is the code a recovery was started with, spent only
	// when the new passkey is registered.
	RecoveryCodeHash []byte `json:"recoveryCodeHash,omitempty"`
}

// passkeyUser adapts a user to what the webauthn library expects.
type passkeyUser struct {
	id          uuid.UUID
	handle      []byte
	name        string
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.handle }
func (u *passkeyUser) WebAuthnName() string                       { return u.name }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.name }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// PowChallenge issues a proof of work for a signup.
func (s *Service) PowChallenge() (PowChallenge, error) {
	return s.pow.create()
}

// BeginSignup reserves nothing yet: it checks the username is free and asks
// the browser to create a passkey for it.
func (s *Service) BeginSignup(ctx context.Context, username string, solution PowSolution) (uuid.UUID, json.RawMessage, error) {
	if !usernamePattern.MatchString(username) {
		return uuid.Nil, nil, api.Errorf(http.StatusUnprocessableEntity, "invalid_username",
			"a username is 3 to 32 letters, digits, dots, dashes or underscores")
	}
	if err := s.pow.verify(ctx, s.q, solution); err != nil {
		return uuid.Nil, nil, err
	}
	if taken, err := s.q.UsernameTaken(ctx, username); err != nil {
		return uuid.Nil, nil, err
	} else if taken {
		return uuid.Nil, nil, api.Errorf(http.StatusConflict, "username_taken", "that username is taken")
	}

	handle := make([]byte, 32)
	if _, err := rand.Read(handle); err != nil {
		return uuid.Nil, nil, err
	}
	user := &passkeyUser{handle: handle, name: username}

	creation, session, err := s.webauthn.BeginRegistration(user, registrationOptions(nil)...)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("begin passkey registration: %w", err)
	}
	id, err := s.storeCeremony(ctx, kindSignup, nil, &username, handle, ceremonyState{WebAuthn: *session})
	if err != nil {
		return uuid.Nil, nil, err
	}
	options, err := json.Marshal(creation.Response)
	return id, options, err
}

// FinishSignup registers the passkey, creates the user with its default
// exercises and recovery codes, and signs it in.
func (s *Service) FinishSignup(ctx context.Context, ceremonyID uuid.UUID, credential json.RawMessage, passkeyName, userAgent string) (User, Session, []string, error) {
	ceremony, state, err := s.takeCeremony(ctx, ceremonyID, kindSignup)
	if err != nil {
		return User{}, Session{}, nil, err
	}
	if ceremony.Username == nil {
		return User{}, Session{}, nil, errCeremony
	}
	pending := &passkeyUser{handle: ceremony.WebauthnHandle, name: *ceremony.Username}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(credential)
	if err != nil {
		return User{}, Session{}, nil, errPasskey
	}
	created, err := s.webauthn.CreateCredential(pending, state.WebAuthn, parsed)
	if err != nil {
		return User{}, Session{}, nil, errPasskey
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return User{}, Session{}, nil, err
	}
	codes, err := newRecoveryCodes()
	if err != nil {
		return User{}, Session{}, nil, err
	}

	var user accountdb.AuthUser
	var session Session
	// The transaction acts as the new user from the start, so the default
	// exercises are written through row level security like any other row.
	err = db.RunInTx(userctx.With(ctx, userID), s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		user, err = q.CreateUser(ctx, accountdb.CreateUserParams{
			ID: userID, Username: *ceremony.Username, WebauthnHandle: ceremony.WebauthnHandle,
		})
		if err != nil {
			return uniqueAs(err, api.Errorf(http.StatusConflict, "username_taken", "that username is taken"))
		}
		if err := s.savePasskey(ctx, q, userID, created, passkeyName); err != nil {
			return err
		}
		if err := saveRecoveryCodes(ctx, q, userID, codes); err != nil {
			return err
		}
		if err := q.SeedExercises(ctx, defaultExercises); err != nil {
			return fmt.Errorf("seed default exercises: %w", err)
		}
		session, err = s.createSession(ctx, q, userID, userAgent)
		return err
	})
	if err != nil {
		return User{}, Session{}, nil, err
	}
	return toUser(user), session, codes, nil
}

// BeginLogin asks the browser for any passkey of this site. No username is
// needed, the passkey carries the user handle.
func (s *Service) BeginLogin(ctx context.Context) (uuid.UUID, json.RawMessage, error) {
	assertion, session, err := s.webauthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("begin passkey login: %w", err)
	}
	id, err := s.storeCeremony(ctx, kindLogin, nil, nil, nil, ceremonyState{WebAuthn: *session})
	if err != nil {
		return uuid.Nil, nil, err
	}
	options, err := json.Marshal(assertion.Response)
	return id, options, err
}

// FinishLogin verifies the passkey and signs its user in.
func (s *Service) FinishLogin(ctx context.Context, ceremonyID uuid.UUID, credential json.RawMessage, userAgent string) (User, Session, error) {
	_, state, err := s.takeCeremony(ctx, ceremonyID, kindLogin)
	if err != nil {
		return User{}, Session{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(credential)
	if err != nil {
		return User{}, Session{}, errPasskey
	}

	var found accountdb.AuthUser
	handler := func(_, userHandle []byte) (webauthn.User, error) {
		user, err := s.q.GetUserByHandle(ctx, userHandle)
		if err != nil {
			return nil, err
		}
		found = user
		return s.loadPasskeyUser(ctx, user)
	}
	_, verified, err := s.webauthn.ValidatePasskeyLogin(handler, state.WebAuthn, parsed)
	if err != nil {
		return User{}, Session{}, errPasskey
	}
	if verified.Authenticator.CloneWarning {
		return User{}, Session{}, errPasskey
	}

	var session Session
	err = db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.TouchPasskey(ctx, accountdb.TouchPasskeyParams{
			CredentialID: verified.ID,
			UserID:       found.ID,
			SignCount:    int64(verified.Authenticator.SignCount),
			BackupState:  verified.Flags.BackupState,
		}); err != nil {
			return err
		}
		session, err = s.createSession(ctx, q, found.ID, userAgent)
		return err
	})
	if err != nil {
		return User{}, Session{}, err
	}
	return toUser(found), session, nil
}

// BeginRecovery starts registering a new passkey for a user who has lost
// theirs, authorized by one of their recovery codes. The code is spent only
// when the passkey is saved.
func (s *Service) BeginRecovery(ctx context.Context, username, code string) (uuid.UUID, json.RawMessage, error) {
	user, err := s.q.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil, errRecovery
	} else if err != nil {
		return uuid.Nil, nil, err
	}
	codeHash := hashRecoveryCode(code)
	usable, err := s.q.RecoveryCodeUsable(ctx, accountdb.RecoveryCodeUsableParams{UserID: user.ID, CodeHash: codeHash})
	if err != nil {
		return uuid.Nil, nil, err
	}
	if !usable {
		return uuid.Nil, nil, errRecovery
	}
	return s.beginRegistration(ctx, kindRecover, user, codeHash)
}

// FinishRecovery saves the new passkey, spends the recovery code and signs
// the user in.
func (s *Service) FinishRecovery(ctx context.Context, ceremonyID uuid.UUID, credential json.RawMessage, passkeyName, userAgent string) (User, Session, error) {
	ceremony, state, err := s.takeCeremony(ctx, ceremonyID, kindRecover)
	if err != nil {
		return User{}, Session{}, err
	}
	if ceremony.UserID == nil {
		return User{}, Session{}, errCeremony
	}
	user, created, err := s.finishRegistration(ctx, *ceremony.UserID, state, credential)
	if err != nil {
		return User{}, Session{}, err
	}

	var session Session
	err = db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		spent, err := q.UseRecoveryCode(ctx, accountdb.UseRecoveryCodeParams{UserID: user.ID, CodeHash: state.RecoveryCodeHash})
		if err != nil {
			return err
		}
		if spent == 0 {
			return errRecovery
		}
		if err := s.savePasskey(ctx, q, user.ID, created, passkeyName); err != nil {
			return err
		}
		session, err = s.createSession(ctx, q, user.ID, userAgent)
		return err
	})
	if err != nil {
		return User{}, Session{}, err
	}
	return toUser(user), session, nil
}

// BeginAddPasskey starts registering another passkey for a signed in user.
func (s *Service) BeginAddPasskey(ctx context.Context, userID uuid.UUID) (uuid.UUID, json.RawMessage, error) {
	user, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return uuid.Nil, nil, err
	}
	return s.beginRegistration(ctx, kindAddPasskey, user, nil)
}

// FinishAddPasskey saves the new passkey.
func (s *Service) FinishAddPasskey(ctx context.Context, userID, ceremonyID uuid.UUID, credential json.RawMessage, name string) (Passkey, error) {
	ceremony, state, err := s.takeCeremony(ctx, ceremonyID, kindAddPasskey)
	if err != nil {
		return Passkey{}, err
	}
	if ceremony.UserID == nil || *ceremony.UserID != userID {
		return Passkey{}, errCeremony
	}
	_, created, err := s.finishRegistration(ctx, userID, state, credential)
	if err != nil {
		return Passkey{}, err
	}
	if err := s.savePasskey(ctx, s.q, userID, created, name); err != nil {
		return Passkey{}, err
	}
	return Passkey{ID: encodeCredentialID(created.ID), Name: passkeyNameOr(name), Created: s.now(), Synced: created.Flags.BackupEligible}, nil
}

// Passkey is one registered passkey as the API shows it.
type Passkey struct {
	ID       string     `json:"id" doc:"The credential id, base64url."`
	Name     string     `json:"name"`
	Created  time.Time  `json:"created"`
	LastUsed *time.Time `json:"lastUsed,omitempty"`
	Synced   bool       `json:"synced" doc:"The passkey can be backed up, e.g. to iCloud Keychain or Google Password Manager."`
}

// ListPasskeys returns the user's passkeys.
func (s *Service) ListPasskeys(ctx context.Context, userID uuid.UUID) ([]Passkey, error) {
	rows, err := s.q.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Passkey, len(rows))
	for i, row := range rows {
		out[i] = Passkey{
			ID:       encodeCredentialID(row.CredentialID),
			Name:     row.Name,
			Created:  row.CreatedAt,
			LastUsed: row.LastUsedAt,
			Synced:   row.BackupEligible,
		}
	}
	return out, nil
}

// RenamePasskey sets a passkey's display name.
func (s *Service) RenamePasskey(ctx context.Context, userID uuid.UUID, id, name string) error {
	credentialID, err := decodeCredentialID(id)
	if err != nil {
		return err
	}
	renamed, err := s.q.RenamePasskey(ctx, accountdb.RenamePasskeyParams{CredentialID: credentialID, UserID: userID, Name: name})
	if err != nil {
		return err
	}
	if renamed == 0 {
		return fmt.Errorf("%w: no such passkey", api.ErrNotFound)
	}
	return nil
}

// DeletePasskey removes a passkey, never the last one.
func (s *Service) DeletePasskey(ctx context.Context, userID uuid.UUID, id string) error {
	credentialID, err := decodeCredentialID(id)
	if err != nil {
		return err
	}
	return db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Lock the user's passkeys so two deletes cannot both see two left.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM auth.passkeys WHERE user_id = $1 FOR UPDATE`, userID); err != nil {
			return err
		}
		count, err := q.CountPasskeys(ctx, userID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return api.Errorf(http.StatusConflict, "last_passkey", "add another passkey before removing this one")
		}
		deleted, err := q.DeletePasskey(ctx, accountdb.DeletePasskeyParams{CredentialID: credentialID, UserID: userID})
		if err != nil {
			return err
		}
		if deleted == 0 {
			return fmt.Errorf("%w: no such passkey", api.ErrNotFound)
		}
		return nil
	})
}

// RegenerateRecoveryCodes replaces every recovery code with a new set.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	codes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	err = db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteRecoveryCodes(ctx, userID); err != nil {
			return err
		}
		return saveRecoveryCodes(ctx, q, userID, codes)
	})
	return codes, err
}

// Me is the signed in user with the numbers the settings page shows.
type Me struct {
	User
	Passkeys          int64 `json:"passkeys"`
	RecoveryCodesLeft int64 `json:"recoveryCodesLeft"`
}

// Me returns the signed in user.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (Me, error) {
	user, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	passkeys, err := s.q.CountPasskeys(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	codes, err := s.q.CountUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	return Me{User: toUser(user), Passkeys: passkeys, RecoveryCodesLeft: codes}, nil
}

// Logout ends a web session.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.q.DeleteSession(ctx, hashToken(token))
}

// DeleteAccount removes the user and, through the foreign keys, everything
// they own.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return s.q.DeleteUser(ctx, userID)
}

// Authenticate resolves a web session token to its user, sliding the expiry.
func (s *Service) Authenticate(ctx context.Context, creds api.Credentials) (uuid.UUID, error) {
	if creds.SessionToken == "" {
		return uuid.Nil, fmt.Errorf("%w: sign in required", api.ErrUnauthorized)
	}
	hash := hashToken(creds.SessionToken)
	session, err := s.q.GetSession(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("%w: the session has expired, sign in again", api.ErrUnauthorized)
	} else if err != nil {
		return uuid.Nil, err
	}
	if s.now().Sub(session.LastSeenAt) > sessionTouchInterval {
		if err := s.q.ExtendSession(ctx, accountdb.ExtendSessionParams{
			TokenHash: hash, ExpiresAt: s.now().Add(s.sessionTTL),
		}); err != nil {
			return uuid.Nil, err
		}
	}
	return session.UserID, nil
}

func (s *Service) beginRegistration(ctx context.Context, kind string, user accountdb.AuthUser, recoveryCodeHash []byte) (uuid.UUID, json.RawMessage, error) {
	existing, err := s.loadPasskeyUser(ctx, user)
	if err != nil {
		return uuid.Nil, nil, err
	}
	creation, session, err := s.webauthn.BeginRegistration(existing, registrationOptions(existing.credentials)...)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("begin passkey registration: %w", err)
	}
	id, err := s.storeCeremony(ctx, kind, &user.ID, nil, nil, ceremonyState{
		WebAuthn:         *session,
		RecoveryCodeHash: recoveryCodeHash,
	})
	if err != nil {
		return uuid.Nil, nil, err
	}
	options, err := json.Marshal(creation.Response)
	return id, options, err
}

func (s *Service) finishRegistration(ctx context.Context, userID uuid.UUID, state ceremonyState, credential json.RawMessage) (accountdb.AuthUser, *webauthn.Credential, error) {
	user, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return accountdb.AuthUser{}, nil, err
	}
	existing, err := s.loadPasskeyUser(ctx, user)
	if err != nil {
		return accountdb.AuthUser{}, nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(credential)
	if err != nil {
		return accountdb.AuthUser{}, nil, errPasskey
	}
	created, err := s.webauthn.CreateCredential(existing, state.WebAuthn, parsed)
	if err != nil {
		return accountdb.AuthUser{}, nil, errPasskey
	}
	return user, created, nil
}

func registrationOptions(existing []webauthn.Credential) []webauthn.RegistrationOption {
	exclusions := make([]protocol.CredentialDescriptor, len(existing))
	for i, credential := range existing {
		exclusions[i] = credential.Descriptor()
	}
	return []webauthn.RegistrationOption{
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithExclusions(exclusions),
	}
}

func (s *Service) loadPasskeyUser(ctx context.Context, user accountdb.AuthUser) (*passkeyUser, error) {
	rows, err := s.q.ListPasskeys(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	credentials := make([]webauthn.Credential, len(rows))
	for i, row := range rows {
		transports := make([]protocol.AuthenticatorTransport, len(row.Transports))
		for j, t := range row.Transports {
			transports[j] = protocol.AuthenticatorTransport(t)
		}
		credentials[i] = webauthn.Credential{
			ID:              row.CredentialID,
			PublicKey:       row.PublicKey,
			AttestationType: row.AttestationType,
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				UserPresent:    true,
				UserVerified:   true,
				BackupEligible: row.BackupEligible,
				BackupState:    row.BackupState,
			},
			Authenticator: webauthn.Authenticator{AAGUID: row.Aaguid, SignCount: uint32(row.SignCount)},
		}
	}
	return &passkeyUser{id: user.ID, handle: user.WebauthnHandle, name: string(user.Username), credentials: credentials}, nil
}

func (s *Service) savePasskey(ctx context.Context, q *accountdb.Queries, userID uuid.UUID, credential *webauthn.Credential, name string) error {
	transports := make([]string, len(credential.Transport))
	for i, t := range credential.Transport {
		transports[i] = string(t)
	}
	err := q.CreatePasskey(ctx, accountdb.CreatePasskeyParams{
		CredentialID:    credential.ID,
		UserID:          userID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Aaguid:          credential.Authenticator.AAGUID,
		SignCount:       int64(credential.Authenticator.SignCount),
		Transports:      transports,
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		Name:            passkeyNameOr(name),
	})
	return uniqueAs(err, api.Errorf(http.StatusConflict, "passkey_exists", "this passkey is already registered"))
}

func saveRecoveryCodes(ctx context.Context, q *accountdb.Queries, userID uuid.UUID, codes []string) error {
	for _, code := range codes {
		if err := q.CreateRecoveryCode(ctx, accountdb.CreateRecoveryCodeParams{UserID: userID, CodeHash: hashRecoveryCode(code)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createSession(ctx context.Context, q *accountdb.Queries, userID uuid.UUID, userAgent string) (Session, error) {
	token, hash, err := newToken()
	if err != nil {
		return Session{}, err
	}
	now := s.now()
	absolute := now.Add(s.sessionAbsoluteTTL)
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	err = q.CreateSession(ctx, accountdb.CreateSessionParams{
		TokenHash:         hash,
		UserID:            userID,
		UserAgent:         userAgent,
		ExpiresAt:         now.Add(s.sessionTTL),
		AbsoluteExpiresAt: absolute,
	})
	return Session{Token: token, Expires: absolute}, err
}

func (s *Service) storeCeremony(ctx context.Context, kind string, userID *uuid.UUID, username *string, handle []byte, state ceremonyState) (uuid.UUID, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return uuid.Nil, err
	}
	return s.q.CreateCeremony(ctx, accountdb.CreateCeremonyParams{
		Kind:           kind,
		UserID:         userID,
		Username:       username,
		WebauthnHandle: handle,
		SessionData:    data,
		ExpiresAt:      s.now().Add(ceremonyTTL),
	})
}

func (s *Service) takeCeremony(ctx context.Context, id uuid.UUID, kind string) (accountdb.AuthWebauthnCeremony, ceremonyState, error) {
	ceremony, err := s.q.TakeCeremony(ctx, accountdb.TakeCeremonyParams{ID: id, Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		return ceremony, ceremonyState{}, errCeremony
	} else if err != nil {
		return ceremony, ceremonyState{}, err
	}
	var state ceremonyState
	if err := json.Unmarshal(ceremony.SessionData, &state); err != nil {
		return ceremony, ceremonyState{}, fmt.Errorf("read ceremony state: %w", err)
	}
	return ceremony, state, nil
}

func toUser(user accountdb.AuthUser) User {
	return User{ID: user.ID, Username: string(user.Username), Created: user.CreatedAt}
}

func passkeyNameOr(name string) string {
	if name == "" {
		return "Passkey"
	}
	if len(name) > 64 {
		return name[:64]
	}
	return name
}
