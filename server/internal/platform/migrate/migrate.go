// Package migrate runs the SQL migrations.
//
// The server and the test provisioner both go through here, so a test database
// is built by the same code, from the same files, as a deployed one.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/zanmato/pgmigrate"

	// Registers the pgx driver under database/sql, which pgmigrate takes.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Dir is where migrations are read from. An explicit path wins, which is how
// the container points at /app/migrations. Otherwise they are resolved from
// this file's location in the source tree.
func Dir(explicit string) string {
	if explicit != "" {
		return explicit
	}
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "db", "migrations")
}

// Up applies every migration that has not run yet.
func Up(ctx context.Context, dsn, dir string, logger *slog.Logger) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = conn.Close() }()

	migrator, err := pgmigrate.NewMigrator(conn, Adapt(logger), dir)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if err := migrator.MigrateUp(ctx); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down rolls back every migration applied after version. A developer tool,
// production is forward only.
func Down(ctx context.Context, dsn, dir string, version int, logger *slog.Logger) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = conn.Close() }()

	migrator, err := pgmigrate.NewMigrator(conn, Adapt(logger), dir)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if err := migrator.MigrateDown(ctx, version); err != nil {
		return fmt.Errorf("migrate down to %d: %w", version, err)
	}
	return nil
}

// Adapt bridges slog to the printf style logger pgmigrate expects.
func Adapt(logger *slog.Logger) pgmigrate.Logger {
	return logAdapter{logger: logger}
}

type logAdapter struct {
	logger *slog.Logger
}

func (l logAdapter) Infof(template string, args ...any) {
	l.logger.Info(fmt.Sprintf(template, args...))
}

func (l logAdapter) Warnf(template string, args ...any) {
	l.logger.Warn(fmt.Sprintf(template, args...))
}
