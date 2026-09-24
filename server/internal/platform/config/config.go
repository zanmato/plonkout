// Package config loads the server configuration from a TOML file.
//
// The file is the single source of configuration. A few secrets and
// connection strings can be overridden from the environment, so a container can
// keep them out of the mounted file.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// PathEnv names the environment variable that points at the config file.
const PathEnv = "PLONKOUT_CONFIG"

// Config is the whole configuration.
type Config struct {
	// Development relaxes checks that only matter for a deployment, such as the
	// length of the secret key.
	Development bool   `toml:"development"`
	LogLevel    string `toml:"log_level"`

	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	Auth     Auth     `toml:"auth"`
	WebAuthn WebAuthn `toml:"webauthn"`
	Frontend Frontend `toml:"frontend"`
	OAuth    OAuth    `toml:"oauth"`
}

// Server is the listener and the public address.
type Server struct {
	Addr string `toml:"addr"`
	// BaseURL is the public origin the app is reached at, e.g.
	// https://plonkout.example. OAuth issuer, MCP resource and the passkey
	// relying party are all derived from it.
	BaseURL string `toml:"base_url"`
}

// Database holds the connection strings.
type Database struct {
	// App is the role the server runs as. It must not bypass row level security.
	App string `toml:"app"`
	// Migrate owns the schema and runs migrations.
	Migrate string `toml:"migrate"`
	// Provision is a superuser connection only the test suite uses, to create
	// template databases.
	Provision string `toml:"provision"`
	// MigrationsPath overrides where migrations are read from. Empty means the
	// source tree, which is right for development and tests.
	MigrationsPath string `toml:"migrations_path"`
	MaxConns       int32  `toml:"max_conns"`
}

// Auth configures sessions and the keys derived from the secret.
type Auth struct {
	// SecretKey signs proof of work challenges and derives other keys. At least
	// 32 bytes outside development.
	SecretKey          string   `toml:"secret_key"`
	SessionTTL         Duration `toml:"session_ttl"`
	SessionAbsoluteTTL Duration `toml:"session_absolute_ttl"`
}

// WebAuthn configures the passkey relying party. Both fields default from the
// base URL.
type WebAuthn struct {
	RPID    string   `toml:"rp_id"`
	RPName  string   `toml:"rp_name"`
	Origins []string `toml:"origins"`
}

// Frontend is where the built SPA is served from. Empty serves no SPA, which
// is what development with the Vite dev server wants.
type Frontend struct {
	Path string `toml:"path"`
}

// OAuth configures the authorization server MCP clients use.
type OAuth struct {
	AccessTokenTTL           Duration `toml:"access_token_ttl"`
	RefreshTokenTTL          Duration `toml:"refresh_token_ttl"`
	AllowDynamicRegistration bool     `toml:"allow_dynamic_registration"`
}

// Duration is a time.Duration written as a string, e.g. "720h". TOML has no
// duration type and time.Duration is not a text unmarshaler.
type Duration time.Duration

// UnmarshalText parses a Go duration string.
func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// Duration returns the value as a time.Duration.
func (d Duration) Duration() time.Duration { return time.Duration(d) }

// Path picks the config file: an explicit path, then PLONKOUT_CONFIG, then
// config.toml in the working directory.
func Path(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if fromEnv := os.Getenv(PathEnv); fromEnv != "" {
		return fromEnv
	}
	return "config.toml"
}

// Load reads, overrides from the environment, defaults and validates.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	meta, err := toml.Decode(string(b), &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, key := range undecoded {
			keys[i] = key.String()
		}
		return nil, fmt.Errorf("config %s has unknown keys: %s", path, strings.Join(keys, ", "))
	}

	cfg.applyEnv(os.Getenv)
	if err := cfg.applyDefaults(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return &cfg, nil
}

// envOverrides are the values a deployment may keep out of the file.
func (c *Config) envOverrides() map[string]*string {
	return map[string]*string{
		"PLONKOUT_SERVER_BASE_URL":          &c.Server.BaseURL,
		"PLONKOUT_DATABASE_APP":             &c.Database.App,
		"PLONKOUT_DATABASE_MIGRATE":         &c.Database.Migrate,
		"PLONKOUT_DATABASE_MIGRATIONS_PATH": &c.Database.MigrationsPath,
		"PLONKOUT_AUTH_SECRET_KEY":          &c.Auth.SecretKey,
		"PLONKOUT_FRONTEND_PATH":            &c.Frontend.Path,
	}
}

func (c *Config) applyEnv(lookup func(string) string) {
	for name, target := range c.envOverrides() {
		if value := lookup(name); value != "" {
			*target = value
		}
	}
}

func (c *Config) applyDefaults() error {
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Database.MaxConns == 0 {
		c.Database.MaxConns = 10
	}
	if c.Auth.SessionTTL == 0 {
		c.Auth.SessionTTL = Duration(30 * 24 * time.Hour)
	}
	if c.Auth.SessionAbsoluteTTL == 0 {
		c.Auth.SessionAbsoluteTTL = Duration(180 * 24 * time.Hour)
	}
	if c.OAuth.AccessTokenTTL == 0 {
		c.OAuth.AccessTokenTTL = Duration(time.Hour)
	}
	if c.OAuth.RefreshTokenTTL == 0 {
		c.OAuth.RefreshTokenTTL = Duration(30 * 24 * time.Hour)
	}

	base, err := url.Parse(c.Server.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return fmt.Errorf("server.base_url %q must be an absolute URL", c.Server.BaseURL)
	}
	c.Server.BaseURL = strings.TrimRight(base.String(), "/")
	if c.WebAuthn.RPID == "" {
		c.WebAuthn.RPID = base.Hostname()
	}
	if c.WebAuthn.RPName == "" {
		c.WebAuthn.RPName = "Plonkout"
	}
	if len(c.WebAuthn.Origins) == 0 {
		c.WebAuthn.Origins = []string{base.Scheme + "://" + base.Host}
	}
	return nil
}

func (c *Config) validate() error {
	var errs []error
	if c.Database.App == "" {
		errs = append(errs, errors.New("database.app is required"))
	}
	if c.Database.Migrate == "" {
		errs = append(errs, errors.New("database.migrate is required"))
	}
	if !c.Development && len(c.Auth.SecretKey) < 32 {
		errs = append(errs, errors.New("auth.secret_key must be at least 32 characters"))
	}
	if c.Auth.SecretKey == "" {
		errs = append(errs, errors.New("auth.secret_key is required"))
	}
	return errors.Join(errs...)
}

// RLSEnforced reports whether the app connects as a different role than the
// one that owns the schema. The owner bypasses row level security, so the same
// DSN for both means the policies protect nothing.
func (c *Config) RLSEnforced() bool {
	return c.Database.App != c.Database.Migrate
}

// Logger builds the process logger at the configured level.
func (c *Config) Logger() *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		level = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// MCPResource is the canonical URL of the MCP endpoint, the audience OAuth
// tokens are bound to.
func (c *Config) MCPResource() string {
	return c.Server.BaseURL + "/mcp"
}
