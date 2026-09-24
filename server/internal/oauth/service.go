// Package oauth is the OAuth 2.1 authorization server MCP clients sign in
// through: Claude on the web, Claude Desktop and Claude Code among them.
//
// It is deliberately small. Authorization code with mandatory PKCE, rotating
// refresh tokens, dynamic client registration and client metadata documents,
// and nothing else: no implicit grant, no password grant, no client
// credentials, because the whole point is that a person approves what an
// agent may do. The person signs in with their passkey in the app and approves
// on its consent page.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/oauth/oauthdb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/config"
)

// OAuth error codes (RFC 6749). They reach the client verbatim, so they never
// carry detail about why something failed.
var (
	ErrInvalidClient        = errors.New("invalid_client")
	ErrInvalidGrant         = errors.New("invalid_grant")
	ErrInvalidRequest       = errors.New("invalid_request")
	ErrUnsupportedGrantType = errors.New("unsupported_grant_type")
	ErrInvalidScope         = errors.New("invalid_scope")
	ErrAccessDenied         = errors.New("access_denied")

	// errRedirectURI marks a refused redirect_uri, which registration reports
	// with its own code.
	errRedirectURI = errors.New("redirect_uri refused")
)

// Scopes. training is everything an agent can do with the user's training
// data. offline_access asks for a refresh token, which every grant gets anyway.
const (
	ScopeTraining      = "training"
	ScopeOfflineAccess = "offline_access"
)

// SupportedScopes is what a client may ask for.
var SupportedScopes = []string{ScopeTraining, ScopeOfflineAccess}

const codeTTL = time.Minute

// Service is the authorization server.
type Service struct {
	q   *oauthdb.Queries
	cfg Config
	// metadata fetches client metadata documents. Replaced in tests.
	metadata *metadataFetcher
	now      func() time.Time
}

// Config configures the authorization server.
type Config struct {
	// Issuer is the public base URL.
	Issuer string
	// Resource is the MCP endpoint's canonical URL, the audience of every token.
	Resource string
	// ConsentPath is where /oauth/authorize sends the browser, a route of the SPA.
	ConsentPath         string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	DynamicRegistration bool
}

// NewService builds the authorization server.
func NewService(pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{
		q: oauthdb.New(pool),
		cfg: Config{
			Issuer:              cfg.Server.BaseURL,
			Resource:            cfg.MCPResource(),
			ConsentPath:         "/#/authorize",
			AccessTokenTTL:      cfg.OAuth.AccessTokenTTL.Duration(),
			RefreshTokenTTL:     cfg.OAuth.RefreshTokenTTL.Duration(),
			DynamicRegistration: cfg.OAuth.AllowDynamicRegistration,
		},
		metadata: newMetadataFetcher(),
		now:      time.Now,
	}
}

// Client is a registered or fetched client.
type Client struct {
	ID           string
	Name         string
	RedirectURIs []string
}

// RegistrationRequest is an RFC 7591 registration.
type RegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

// RegistrationResponse answers a registration.
type RegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// Register implements dynamic client registration. Every client is public and
// proves itself with PKCE, since MCP clients run on people's machines and
// cannot keep a secret. A registration grants nothing on its own: every
// authority a client ends up with was approved by a person.
func (s *Service) Register(ctx context.Context, req RegistrationRequest) (*RegistrationResponse, error) {
	if !s.cfg.DynamicRegistration {
		return nil, fmt.Errorf("%w: dynamic registration is disabled", ErrInvalidRequest)
	}
	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > 10 {
		return nil, fmt.Errorf("%w: between one and ten redirect_uris are required", ErrInvalidRequest)
	}
	for _, redirect := range req.RedirectURIs {
		if err := validateRedirectURI(redirect); err != nil {
			return nil, err
		}
	}
	for _, grant := range req.GrantTypes {
		if grant != "authorization_code" && grant != "refresh_token" {
			return nil, fmt.Errorf("%w: unsupported grant type %q", ErrInvalidRequest, grant)
		}
	}
	if len(req.ClientName) > 200 {
		req.ClientName = req.ClientName[:200]
	}

	id, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	id = "dcr_" + id
	metadata, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertClient(ctx, oauthdb.UpsertClientParams{
		ClientID: id, Kind: "dcr", ClientName: req.ClientName, RedirectUris: req.RedirectURIs, Metadata: metadata,
	})
	if err != nil {
		return nil, err
	}
	return &RegistrationResponse{
		ClientID:                row.ClientID,
		ClientIDIssuedAt:        row.CreatedAt.Unix(),
		ClientName:              row.ClientName,
		RedirectURIs:            row.RedirectUris,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	}, nil
}

// client looks a client up, fetching and caching the metadata document of an
// https client_id when it is not known or its cache has run out.
func (s *Service) client(ctx context.Context, id string) (*Client, error) {
	row, err := s.q.GetClient(ctx, id)
	switch {
	case err == nil && (row.Kind == "dcr" || (row.CacheUntil != nil && row.CacheUntil.After(s.now()))):
		return &Client{ID: row.ClientID, Name: row.ClientName, RedirectURIs: row.RedirectUris}, nil
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	case !isMetadataURL(id):
		return nil, ErrInvalidClient
	}

	doc, maxAge, err := s.metadata.fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidClient, err)
	}
	cacheUntil := s.now().Add(maxAge)
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	stored, err := s.q.UpsertClient(ctx, oauthdb.UpsertClientParams{
		ClientID: id, Kind: "cimd", ClientName: doc.ClientName, RedirectUris: doc.RedirectURIs,
		Metadata: raw, CacheUntil: &cacheUntil,
	})
	if err != nil {
		return nil, err
	}
	return &Client{ID: stored.ClientID, Name: stored.ClientName, RedirectURIs: stored.RedirectUris}, nil
}

// AuthorizationRequest is a validated /authorize request, carried to the
// consent page and back in the query string.
type AuthorizationRequest struct {
	ClientID      string `json:"clientId"`
	ClientName    string `json:"clientName"`
	RedirectURI   string `json:"redirectUri"`
	State         string `json:"state,omitempty"`
	Scope         string `json:"scope"`
	CodeChallenge string `json:"codeChallenge"`
	Resource      string `json:"resource"`
}

// RedirectError is a failure that may be reported to the client by
// redirecting, because the client and redirect_uri were valid. Anything else
// is shown to the person instead, since the redirect could not be trusted.
type RedirectError struct {
	Err         error
	RedirectURI string
	State       string
}

func (e *RedirectError) Error() string { return e.Err.Error() }
func (e *RedirectError) Unwrap() error { return e.Err }

// Prepare validates an /authorize request. It decides nothing: the person
// does, on the consent page.
func (s *Service) Prepare(ctx context.Context, query url.Values) (*AuthorizationRequest, error) {
	clientID := query.Get("client_id")
	if clientID == "" {
		return nil, fmt.Errorf("%w: client_id is required", ErrInvalidRequest)
	}
	client, err := s.client(ctx, clientID)
	if err != nil {
		return nil, err
	}
	redirectURI := query.Get("redirect_uri")
	if !redirectAllowed(client.RedirectURIs, redirectURI) {
		return nil, fmt.Errorf("%w: redirect_uri is not registered for this client", ErrInvalidRequest)
	}

	fail := func(err error) error {
		return &RedirectError{Err: err, RedirectURI: redirectURI, State: query.Get("state")}
	}
	if query.Get("response_type") != "code" {
		return nil, fail(fmt.Errorf("%w: only the authorization code flow is supported", ErrInvalidRequest))
	}
	challenge := query.Get("code_challenge")
	if challenge == "" || query.Get("code_challenge_method") != "S256" {
		return nil, fail(fmt.Errorf("%w: PKCE with S256 is required", ErrInvalidRequest))
	}
	if !s.resourceMatches(query.Get("resource")) {
		return nil, fail(fmt.Errorf("%w: unknown resource", ErrInvalidRequest))
	}
	scope, err := normalizeScope(query.Get("scope"))
	if err != nil {
		return nil, fail(err)
	}

	return &AuthorizationRequest{
		ClientID:      client.ID,
		ClientName:    client.Name,
		RedirectURI:   redirectURI,
		State:         query.Get("state"),
		Scope:         scope,
		CodeChallenge: challenge,
		Resource:      s.cfg.Resource,
	}, nil
}

// ConsentURL is where /authorize sends the browser.
func (s *Service) ConsentURL(query url.Values) string {
	return s.cfg.Issuer + s.cfg.ConsentPath + "?" + query.Encode()
}

// Decide records the person's answer on the consent page and returns where to
// send the browser: back to the client with a code, or with access_denied.
func (s *Service) Decide(ctx context.Context, userID uuid.UUID, query url.Values, approved bool) (string, error) {
	req, err := s.Prepare(ctx, query)
	if err != nil {
		return "", err
	}
	if !approved {
		return s.redirect(req.RedirectURI, url.Values{"error": {"access_denied"}, "state": {req.State}}), nil
	}

	code, err := randomToken(32)
	if err != nil {
		return "", err
	}
	if err := s.q.CreateCode(ctx, oauthdb.CreateCodeParams{
		CodeHash:      hashToken(code),
		ClientID:      req.ClientID,
		UserID:        userID,
		RedirectUri:   req.RedirectURI,
		CodeChallenge: req.CodeChallenge,
		Scope:         req.Scope,
		Resource:      req.Resource,
		ExpiresAt:     s.now().Add(codeTTL),
	}); err != nil {
		return "", err
	}
	return s.redirect(req.RedirectURI, url.Values{"code": {code}, "state": {req.State}}), nil
}

// redirect adds parameters to a redirect URI. iss says which server answered
// (RFC 9207), so a client talking to several cannot be tricked into mixing
// them up.
func (s *Service) redirect(redirectURI string, params url.Values) string {
	target, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	values := target.Query()
	for key, vals := range params {
		if len(vals) > 0 && vals[0] != "" {
			values.Set(key, vals[0])
		}
	}
	values.Set("iss", s.cfg.Issuer)
	target.RawQuery = values.Encode()
	return target.String()
}

// TokenResponse answers the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

// ExchangeCode implements the authorization_code grant.
func (s *Service) ExchangeCode(ctx context.Context, form url.Values) (*TokenResponse, error) {
	code, verifier := form.Get("code"), form.Get("code_verifier")
	if code == "" || verifier == "" {
		return nil, fmt.Errorf("%w: code and code_verifier are required", ErrInvalidRequest)
	}
	stored, err := s.q.TakeCode(ctx, hashToken(code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidGrant
	} else if err != nil {
		return nil, err
	}
	if stored.ClientID != form.Get("client_id") || stored.RedirectUri != form.Get("redirect_uri") {
		return nil, ErrInvalidGrant
	}
	if resource := form.Get("resource"); resource != "" && !s.resourceMatches(resource) {
		return nil, ErrInvalidGrant
	}
	if !verifyPKCE(stored.CodeChallenge, verifier) {
		return nil, ErrInvalidGrant
	}

	refresh, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	grant, err := s.q.CreateGrant(ctx, oauthdb.CreateGrantParams{
		ClientID:         stored.ClientID,
		UserID:           stored.UserID,
		Scope:            stored.Scope,
		Resource:         stored.Resource,
		RefreshHash:      hashToken(refresh),
		RefreshExpiresAt: s.now().Add(s.cfg.RefreshTokenTTL),
	})
	if err != nil {
		return nil, err
	}
	return s.issue(ctx, grant, refresh)
}

// Refresh implements the refresh_token grant, rotating the token every time.
// A token that was already rotated away revokes the whole grant: either the
// client or a thief holds a copy, and there is no telling which.
func (s *Service) Refresh(ctx context.Context, form url.Values) (*TokenResponse, error) {
	old := form.Get("refresh_token")
	if old == "" {
		return nil, fmt.Errorf("%w: refresh_token is required", ErrInvalidRequest)
	}
	next, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	grant, err := s.q.RotateRefreshToken(ctx, oauthdb.RotateRefreshTokenParams{
		OldHash:          hashToken(old),
		NewHash:          hashToken(next),
		RefreshExpiresAt: s.now().Add(s.cfg.RefreshTokenTTL),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := s.q.RevokeGrantByPreviousRefresh(ctx, hashToken(old)); err != nil {
			return nil, err
		}
		return nil, ErrInvalidGrant
	} else if err != nil {
		return nil, err
	}
	if client := form.Get("client_id"); client != "" && client != grant.ClientID {
		return nil, ErrInvalidGrant
	}
	return s.issue(ctx, grant, next)
}

func (s *Service) issue(ctx context.Context, grant oauthdb.AuthOauthGrant, refresh string) (*TokenResponse, error) {
	access, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	if err := s.q.CreateAccessToken(ctx, oauthdb.CreateAccessTokenParams{
		TokenHash: hashToken(access), GrantID: grant.ID, ExpiresAt: s.now().Add(s.cfg.AccessTokenTTL),
	}); err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()),
		RefreshToken: refresh,
		Scope:        grant.Scope,
	}, nil
}

// Revoke implements RFC 7009. It answers the same whether or not the token
// existed, so it cannot be used to probe for valid tokens.
func (s *Service) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	hash := hashToken(token)
	if err := s.q.RevokeAccessToken(ctx, hash); err != nil {
		return err
	}
	return s.q.RevokeGrantByRefresh(ctx, hash)
}

// TokenInfo is a verified access token.
type TokenInfo struct {
	UserID  uuid.UUID
	GrantID uuid.UUID
	Scopes  []string
	Expires time.Time
}

// Verify checks an access token: known, not expired, its grant not revoked,
// and issued for this server's MCP resource.
func (s *Service) Verify(ctx context.Context, token string) (*TokenInfo, error) {
	row, err := s.q.VerifyAccessToken(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: the access token is not valid", api.ErrUnauthorized)
	} else if err != nil {
		return nil, err
	}
	if !s.resourceMatches(row.Resource) {
		return nil, fmt.Errorf("%w: the access token is for another resource", api.ErrUnauthorized)
	}
	if err := s.q.TouchGrant(ctx, row.GrantID); err != nil {
		return nil, err
	}
	return &TokenInfo{UserID: row.UserID, GrantID: row.GrantID, Scopes: strings.Fields(row.Scope), Expires: row.ExpiresAt}, nil
}

// ConnectedApp is a grant as the settings page shows it.
type ConnectedApp struct {
	ID       uuid.UUID  `json:"id"`
	ClientID string     `json:"clientId"`
	Name     string     `json:"name"`
	Scope    string     `json:"scope"`
	Created  time.Time  `json:"created"`
	LastUsed *time.Time `json:"lastUsed"`
}

// ConnectedApps lists what the user has approved.
func (s *Service) ConnectedApps(ctx context.Context, userID uuid.UUID) ([]ConnectedApp, error) {
	rows, err := s.q.ListGrants(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]ConnectedApp, len(rows))
	for i, row := range rows {
		name := row.ClientName
		if name == "" {
			name = row.ClientID
		}
		out[i] = ConnectedApp{
			ID: row.ID, ClientID: row.ClientID, Name: name, Scope: row.Scope,
			Created: row.CreatedAt, LastUsed: row.LastUsedAt,
		}
	}
	return out, nil
}

// Disconnect revokes one of the user's grants and every token of it.
func (s *Service) Disconnect(ctx context.Context, userID, grantID uuid.UUID) error {
	revoked, err := s.q.RevokeGrant(ctx, oauthdb.RevokeGrantParams{ID: grantID, UserID: userID})
	if err != nil {
		return err
	}
	if revoked == 0 {
		return fmt.Errorf("%w: no such connected app", api.ErrNotFound)
	}
	return nil
}

// resourceMatches compares a requested resource with this server's, allowing
// an empty one (the default) and ignoring case in the host and a trailing slash.
func (s *Service) resourceMatches(resource string) bool {
	if resource == "" {
		return true
	}
	return strings.EqualFold(strings.TrimRight(resource, "/"), strings.TrimRight(s.cfg.Resource, "/"))
}

func normalizeScope(requested string) (string, error) {
	scopes := []string{}
	for scope := range strings.FieldsSeq(requested) {
		if !slices.Contains(SupportedScopes, scope) {
			return "", fmt.Errorf("%w: unknown scope %q", ErrInvalidScope, scope)
		}
		if !slices.Contains(scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	if !slices.Contains(scopes, ScopeTraining) {
		scopes = append([]string{ScopeTraining}, scopes...)
	}
	return strings.Join(scopes, " "), nil
}

// validateRedirectURI refuses anything that could deliver a code somewhere
// it should not go: plain http is allowed only on loopback, where native
// clients listen.
func validateRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("%w: %w: %q is not an absolute URL without a fragment", ErrInvalidRequest, errRedirectURI, raw)
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopback(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("%w: %w: http is only allowed on loopback", ErrInvalidRequest, errRedirectURI)
	}
	return fmt.Errorf("%w: %w: it must be https, or http on loopback", ErrInvalidRequest, errRedirectURI)
}

// redirectAllowed matches a redirect exactly, except that a loopback redirect
// may use any port: a native client picks a free port per sign in (RFC 8252).
func redirectAllowed(registered []string, redirect string) bool {
	if slices.Contains(registered, redirect) {
		return true
	}
	want, err := url.Parse(redirect)
	if err != nil || want.Scheme != "http" || !isLoopback(want.Hostname()) {
		return false
	}
	for _, r := range registered {
		have, err := url.Parse(r)
		if err != nil || have.Scheme != "http" || !isLoopback(have.Hostname()) {
			continue
		}
		if have.Hostname() == want.Hostname() && have.Path == want.Path && have.RawQuery == want.RawQuery {
			return true
		}
	}
	return false
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// verifyPKCE checks the S256 challenge in constant time.
func verifyPKCE(challenge, verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// statusFor maps an OAuth error to its HTTP status.
func statusFor(err error) int {
	switch {
	case errors.Is(err, ErrInvalidClient):
		return http.StatusUnauthorized
	case errors.Is(err, ErrInvalidGrant), errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrUnsupportedGrantType), errors.Is(err, ErrInvalidScope):
		return http.StatusBadRequest
	case errors.Is(err, ErrAccessDenied):
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

// oauthCode is the RFC 6749 error code of an error.
func oauthCode(err error) string {
	for _, known := range []error{ErrInvalidClient, ErrInvalidGrant, ErrInvalidRequest, ErrUnsupportedGrantType, ErrInvalidScope, ErrAccessDenied} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return "server_error"
}
