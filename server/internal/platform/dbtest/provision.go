package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
	"github.com/zanmato/pgmigrate"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/migrate"
)

// ConfigEnv points the suite at another config file. Without it the committed
// config.test.toml next to go.mod is used.
const ConfigEnv = "PLONKOUT_TEST_CONFIG"

// provisioner migrates one template per migration set and clones a fresh
// database from it per test. Cloning is a file copy inside Postgres, which is
// what makes a real database per test affordable.
type provisioner struct {
	cfg *config.Config
	mtx sync.Mutex
}

var (
	shared     *provisioner
	sharedOnce sync.Once
)

func sharedProvisioner() *provisioner {
	sharedOnce.Do(func() {
		path := os.Getenv(ConfigEnv)
		if path == "" {
			_, filename, _, _ := runtime.Caller(0)
			path = filepath.Join(filepath.Dir(filename), "..", "..", "..", "config.test.toml")
		}
		cfg, err := config.Load(path)
		if err != nil {
			panic(fmt.Errorf("load the test config: %w", err))
		}
		shared = &provisioner{cfg: cfg}
	})
	return shared
}

// clone creates one ephemeral database from the template and returns its name.
func (p *provisioner) clone(tb testing.TB) string {
	tb.Helper()

	conf, err := pgtestdbConfig(p.cfg.Database.Provision, p.cfg.Database.Migrate)
	if err != nil {
		tb.Fatalf("test database configuration: %v", err)
	}
	migrator := &pgMigrator{dir: migrate.Dir(p.cfg.Database.MigrationsPath)}

	// Racing the first template migration produces duplicate create failures,
	// so the first caller through builds it.
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return pgtestdb.Custom(tb, conf, migrator).Database
}

// pgtestdbConfig describes the server to pgtestdb. The provision role is a
// superuser, needed to mark a template. The databases are owned and migrated by
// the migrate role, exactly like a deployed one.
func pgtestdbConfig(provisionDSN, migrateDSN string) (pgtestdb.Config, error) {
	if provisionDSN == "" {
		return pgtestdb.Config{}, fmt.Errorf("the test config has no database.provision connection")
	}
	conf, err := pgxpool.ParseConfig(provisionDSN)
	if err != nil {
		return pgtestdb.Config{}, fmt.Errorf("parse the provision DSN: %w", err)
	}
	owner, err := pgxpool.ParseConfig(migrateDSN)
	if err != nil {
		return pgtestdb.Config{}, fmt.Errorf("parse the migrate DSN: %w", err)
	}

	options := url.Values{}
	for k, v := range conf.ConnConfig.RuntimeParams {
		options.Add(k, v)
	}

	return pgtestdb.Config{
		DriverName:                "pgx",
		User:                      conf.ConnConfig.User,
		Password:                  conf.ConnConfig.Password,
		Host:                      conf.ConnConfig.Host,
		Port:                      strconv.FormatUint(uint64(conf.ConnConfig.Port), 10),
		Database:                  conf.ConnConfig.Database,
		Options:                   options.Encode(),
		TestRole:                  &pgtestdb.Role{Username: owner.ConnConfig.User, Password: owner.ConnConfig.Password},
		ForceTerminateConnections: true,
	}, nil
}

// onDatabase points a DSN at another database on the same server.
func onDatabase(dsn, database string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse DSN: %w", err)
	}
	parsed.Path = "/" + database
	return parsed.String(), nil
}

// pgMigrator runs the repository's migrations into a pgtestdb template.
type pgMigrator struct {
	dir string
}

// Hash keys the template on migration names and contents, so editing a
// migration in place builds a new template instead of reusing the old schema.
func (m *pgMigrator) Hash() (string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return "", err
	}
	hash := common.NewRecursiveHash()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(m.dir, name))
		if err != nil {
			return "", err
		}
		hash.Add([]byte(name))
		hash.Add(contents)
	}
	return hash.String(), nil
}

// Migrate applies the migrations to the template.
func (m *pgMigrator) Migrate(ctx context.Context, conn *sql.DB, _ pgtestdb.Config) error {
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	migrator, err := pgmigrate.NewMigrator(conn, migrate.Adapt(quiet), m.dir)
	if err != nil {
		return err
	}
	return migrator.MigrateUp(ctx)
}
