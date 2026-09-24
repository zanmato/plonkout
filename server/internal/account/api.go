package account

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// CeremonyStart is the first half of a passkey ceremony.
type CeremonyStart struct {
	Ceremony uuid.UUID `json:"ceremony" doc:"Pass back to the matching finish operation."`
	// Options is PublicKeyCredentialCreationOptionsJSON or
	// PublicKeyCredentialRequestOptionsJSON, ready for
	// PublicKeyCredential.parseCreationOptionsFromJSON or
	// parseRequestOptionsFromJSON.
	Options json.RawMessage `json:"options" doc:"WebAuthn options in their JSON form, for PublicKeyCredential.parse*OptionsFromJSON."`
}

// CeremonyFinish is the browser's answer to a ceremony.
type CeremonyFinish struct {
	Ceremony uuid.UUID `json:"ceremony"`
	// Credential is PublicKeyCredential.toJSON().
	Credential json.RawMessage `json:"credential" doc:"The result of PublicKeyCredential.toJSON()."`
}

// withSession answers with only a cookie. Huma reads headers from the output
// struct's own fields, not from embedded ones, so outputs that also carry a
// body declare the field themselves.
type withSession struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

// SignedIn answers a successful sign in.
type SignedIn struct {
	User User `json:"user"`
	// RecoveryCodes are shown once, at signup.
	RecoveryCodes []string `json:"recoveryCodes,omitempty"`
}

// Register declares the account operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"account"}

	api.Register(reg, api.Op{
		ID: "get-signup-challenge", Method: http.MethodGet, Path: "/auth/signup/challenge",
		Summary:     "Issue a proof of work for a signup",
		Description: "The browser brute forces the number in a worker and sends the solution with begin-signup.",
		Tags:        tags, Public: true,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body PowChallenge }, error) {
		challenge, err := s.PowChallenge()
		if err != nil {
			return nil, err
		}
		return &struct{ Body PowChallenge }{Body: challenge}, nil
	})

	type beginSignupBody struct {
		Username string `json:"username" minLength:"3" maxLength:"32" pattern:"^[A-Za-z0-9_.-]+$"`
		// Website is a honeypot. People never see the field, scripts fill it.
		Website string      `json:"website,omitempty" doc:"Leave empty."`
		Pow     PowSolution `json:"pow"`
	}
	api.Register(reg, api.Op{
		ID: "begin-signup", Method: http.MethodPost, Path: "/auth/signup/begin",
		Summary: "Start creating an account with a passkey",
		Tags:    tags, Public: true,
		Errors: []int{http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body beginSignupBody }) (*struct{ Body CeremonyStart }, error) {
		if in.Body.Website != "" {
			return nil, api.Errorf(http.StatusUnprocessableEntity, "signup_refused",
				"the signup could not be verified, reload and try again")
		}
		id, options, err := s.BeginSignup(ctx, in.Body.Username, in.Body.Pow)
		if err != nil {
			return nil, err
		}
		return &struct{ Body CeremonyStart }{Body: CeremonyStart{Ceremony: id, Options: options}}, nil
	})

	type finishWithName struct {
		CeremonyFinish
		PasskeyName string `json:"passkeyName,omitempty" maxLength:"64" doc:"A name that helps tell passkeys apart, e.g. the device."`
	}
	type signedInOutput struct {
		SetCookie http.Cookie `header:"Set-Cookie"`
		Body      SignedIn
	}

	api.Register(reg, api.Op{
		ID: "finish-signup", Method: http.MethodPost, Path: "/auth/signup/finish",
		Summary:     "Save the passkey, create the account and sign in",
		Description: "Answers with the recovery codes, which are never shown again.",
		Tags:        tags, Public: true,
		Errors: []int{http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		UserAgent string `header:"User-Agent"`
		Body      finishWithName
	}) (*signedInOutput, error) {
		user, session, codes, err := s.FinishSignup(ctx, in.Body.Ceremony, in.Body.Credential, in.Body.PasskeyName, in.UserAgent)
		if err != nil {
			return nil, err
		}
		return &signedInOutput{
			SetCookie: sessionCookie(session.Token, session.Expires),
			Body:      SignedIn{User: user, RecoveryCodes: codes},
		}, nil
	})

	api.Register(reg, api.Op{
		ID: "begin-login", Method: http.MethodPost, Path: "/auth/login/begin",
		Summary: "Start signing in with a passkey",
		Tags:    tags, Public: true,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body CeremonyStart }, error) {
		id, options, err := s.BeginLogin(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body CeremonyStart }{Body: CeremonyStart{Ceremony: id, Options: options}}, nil
	})

	api.Register(reg, api.Op{
		ID: "finish-login", Method: http.MethodPost, Path: "/auth/login/finish",
		Summary: "Verify the passkey and sign in",
		Tags:    tags, Public: true,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		UserAgent string `header:"User-Agent"`
		Body      CeremonyFinish
	}) (*signedInOutput, error) {
		user, session, err := s.FinishLogin(ctx, in.Body.Ceremony, in.Body.Credential, in.UserAgent)
		if err != nil {
			return nil, err
		}
		return &signedInOutput{
			SetCookie: sessionCookie(session.Token, session.Expires),
			Body:      SignedIn{User: user},
		}, nil
	})

	type beginRecoveryBody struct {
		Username     string `json:"username" minLength:"1" maxLength:"32"`
		RecoveryCode string `json:"recoveryCode" minLength:"1" maxLength:"32"`
	}
	api.Register(reg, api.Op{
		ID: "begin-recovery", Method: http.MethodPost, Path: "/auth/recover/begin",
		Summary: "Start adding a new passkey with a recovery code",
		Tags:    tags, Public: true,
	}, func(ctx context.Context, in *struct{ Body beginRecoveryBody }) (*struct{ Body CeremonyStart }, error) {
		id, options, err := s.BeginRecovery(ctx, in.Body.Username, in.Body.RecoveryCode)
		if err != nil {
			return nil, err
		}
		return &struct{ Body CeremonyStart }{Body: CeremonyStart{Ceremony: id, Options: options}}, nil
	})

	api.Register(reg, api.Op{
		ID: "finish-recovery", Method: http.MethodPost, Path: "/auth/recover/finish",
		Summary: "Save the new passkey, spend the recovery code and sign in",
		Tags:    tags, Public: true,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		UserAgent string `header:"User-Agent"`
		Body      finishWithName
	}) (*signedInOutput, error) {
		user, session, err := s.FinishRecovery(ctx, in.Body.Ceremony, in.Body.Credential, in.Body.PasskeyName, in.UserAgent)
		if err != nil {
			return nil, err
		}
		return &signedInOutput{
			SetCookie: sessionCookie(session.Token, session.Expires),
			Body:      SignedIn{User: user},
		}, nil
	})

	api.Register(reg, api.Op{
		ID: "logout", Method: http.MethodPost, Path: "/auth/logout",
		Summary: "Sign out of this device",
		Tags:    tags, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *struct {
		Session string `cookie:"__Host-plonkout_sid"`
	}) (*withSession, error) {
		if in.Session != "" {
			if err := s.Logout(ctx, in.Session); err != nil {
				return nil, err
			}
		}
		return &withSession{SetCookie: clearedSessionCookie()}, nil
	})

	api.Register(reg, api.Op{
		ID: "get-me", Method: http.MethodGet, Path: "/account",
		Summary: "The signed in user",
		Tags:    tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body Me }, error) {
		me, err := s.Me(ctx, api.UserID(ctx))
		if err != nil {
			return nil, err
		}
		return &struct{ Body Me }{Body: me}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-account", Method: http.MethodDelete, Path: "/account",
		Summary:     "Delete the account and everything in it",
		Description: "Irreversible. Workouts, plans, templates, settings and passkeys are all removed.",
		Tags:        tags, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, _ *struct{}) (*withSession, error) {
		if err := s.DeleteAccount(ctx, api.UserID(ctx)); err != nil {
			return nil, err
		}
		return &withSession{SetCookie: clearedSessionCookie()}, nil
	})

	api.Register(reg, api.Op{
		ID: "list-passkeys", Method: http.MethodGet, Path: "/account/passkeys",
		Summary: "The signed in user's passkeys",
		Tags:    tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []Passkey }, error) {
		passkeys, err := s.ListPasskeys(ctx, api.UserID(ctx))
		if err != nil {
			return nil, err
		}
		return &struct{ Body []Passkey }{Body: passkeys}, nil
	})

	api.Register(reg, api.Op{
		ID: "begin-add-passkey", Method: http.MethodPost, Path: "/account/passkeys/begin",
		Summary: "Start registering another passkey",
		Tags:    tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body CeremonyStart }, error) {
		id, options, err := s.BeginAddPasskey(ctx, api.UserID(ctx))
		if err != nil {
			return nil, err
		}
		return &struct{ Body CeremonyStart }{Body: CeremonyStart{Ceremony: id, Options: options}}, nil
	})

	api.Register(reg, api.Op{
		ID: "finish-add-passkey", Method: http.MethodPost, Path: "/account/passkeys/finish",
		Summary: "Save another passkey",
		Tags:    tags, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body finishWithName }) (*struct{ Body Passkey }, error) {
		passkey, err := s.FinishAddPasskey(ctx, api.UserID(ctx), in.Body.Ceremony, in.Body.Credential, in.Body.PasskeyName)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Passkey }{Body: passkey}, nil
	})

	type renameBody struct {
		Name string `json:"name" minLength:"1" maxLength:"64"`
	}
	api.Register(reg, api.Op{
		ID: "rename-passkey", Method: http.MethodPatch, Path: "/account/passkeys/{id}",
		Summary: "Rename a passkey",
		Tags:    tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body renameBody
	}) (*struct{}, error) {
		return nil, s.RenamePasskey(ctx, api.UserID(ctx), in.ID, in.Body.Name)
	})

	api.Register(reg, api.Op{
		ID: "delete-passkey", Method: http.MethodDelete, Path: "/account/passkeys/{id}",
		Summary: "Remove a passkey",
		Description: "The last passkey cannot be removed, since the account could not be signed in to again " +
			"without spending a recovery code.",
		Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		return nil, s.DeletePasskey(ctx, api.UserID(ctx), in.ID)
	})

	api.Register(reg, api.Op{
		ID: "regenerate-recovery-codes", Method: http.MethodPost, Path: "/account/recovery-codes",
		Summary:     "Replace every recovery code with a new set",
		Description: "The old codes stop working. The new ones are shown only in this answer.",
		Tags:        tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []string }, error) {
		codes, err := s.RegenerateRecoveryCodes(ctx, api.UserID(ctx))
		if err != nil {
			return nil, err
		}
		return &struct{ Body []string }{Body: codes}, nil
	})
}

// sessionCookie carries the session token. __Host- pins it to this exact
// origin, and HttpOnly keeps it out of reach of any script.
func sessionCookie(token string, expires time.Time) http.Cookie {
	return http.Cookie{
		Name:     api.SessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearedSessionCookie() http.Cookie {
	return http.Cookie{
		Name:     api.SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
