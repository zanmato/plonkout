// Package dbtest gives tests a real, freshly migrated database.
//
// Tests connect as the application role, through the same pool constructor the
// server uses, so row level security applies to them exactly as it does in
// production. The owner pool exists to seed and inspect across users.
package dbtest

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/platform/userctx"
)

// DB is one test's database.
type DB struct {
	// App connects as the application role, row level security applies.
	App *pgxpool.Pool
	// Owner connects as the schema owner, which bypasses row level security.
	Owner *pgxpool.Pool
}

// New clones a fresh database for the test.
func New(tb testing.TB) *DB {
	tb.Helper()

	p := sharedProvisioner()
	database := p.clone(tb)
	logger := slog.New(slog.DiscardHandler)

	appDSN, err := onDatabase(p.cfg.Database.App, database)
	if err != nil {
		tb.Fatal(err)
	}
	ownerDSN, err := onDatabase(p.cfg.Database.Migrate, database)
	if err != nil {
		tb.Fatal(err)
	}

	app, err := db.Open(tb.Context(), appDSN, 4, logger)
	if err != nil {
		tb.Fatalf("open the app pool: %v", err)
	}
	tb.Cleanup(app.Close)

	owner, err := db.Open(tb.Context(), ownerDSN, 2, logger)
	if err != nil {
		tb.Fatalf("open the owner pool: %v", err)
	}
	tb.Cleanup(owner.Close)

	return &DB{App: app, Owner: owner}
}

// NewUser creates a user and returns its id.
func (d *DB) NewUser(tb testing.TB) uuid.UUID {
	tb.Helper()

	handle := make([]byte, 32)
	_, _ = rand.Read(handle)
	username := "user_" + uuid.Must(uuid.NewV4()).String()[:8]

	var id uuid.UUID
	err := d.App.QueryRow(tb.Context(),
		`INSERT INTO auth.users (username, webauthn_handle) VALUES ($1, $2) RETURNING id`,
		username, handle).Scan(&id)
	if err != nil {
		tb.Fatalf("create a user: %v", err)
	}
	return id
}

// As returns a context acting for the user.
func As(ctx context.Context, userID uuid.UUID) context.Context {
	return userctx.With(ctx, userID)
}

// AssertIsolated proves the policy on table: user A sees exactly its own rows,
// cannot read, update or delete user B's rows by naming them, and cannot insert
// a row owned by B. The table must hold rows for exactly those two users.
func (d *DB) AssertIsolated(tb testing.TB, table string, a, b uuid.UUID) {
	tb.Helper()
	ctx := tb.Context()
	name := pgx.Identifier{table}.Sanitize()

	var owned int64
	if err := d.Owner.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE user_id = $1`, name), a).Scan(&owned); err != nil {
		tb.Fatalf("count %s rows of user A: %v", table, err)
	}
	var theirs int64
	if err := d.Owner.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE user_id = $1`, name), b).Scan(&theirs); err != nil {
		tb.Fatalf("count %s rows of user B: %v", table, err)
	}
	if owned == 0 || theirs == 0 {
		tb.Fatalf("%s: seed rows for both users before asserting isolation (A %d, B %d)", table, owned, theirs)
	}

	// One connection for the whole assertion, acting as A.
	conn, err := d.App.Acquire(As(ctx, a))
	if err != nil {
		tb.Fatalf("acquire a connection as user A: %v", err)
	}
	defer conn.Release()

	var visible int64
	if err := conn.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s`, name)).Scan(&visible); err != nil {
		tb.Fatalf("count %s: %v", table, err)
	}
	if visible != owned {
		tb.Fatalf("%s: user A sees %d rows and owns %d", table, visible, owned)
	}

	if err := conn.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE user_id = $1`, name), b).Scan(&visible); err != nil {
		tb.Fatalf("count user B's rows in %s: %v", table, err)
	}
	if visible != 0 {
		tb.Fatalf("%s: user A can read %d rows of user B", table, visible)
	}

	tag, err := conn.Exec(ctx, fmt.Sprintf(`UPDATE %s SET user_id = user_id WHERE user_id = $1`, name), b)
	if err != nil {
		tb.Fatalf("update user B's rows in %s: %v", table, err)
	}
	if tag.RowsAffected() != 0 {
		tb.Fatalf("%s: user A updated %d rows of user B", table, tag.RowsAffected())
	}

	tag, err = conn.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1`, name), b)
	if err != nil {
		tb.Fatalf("delete user B's rows in %s: %v", table, err)
	}
	if tag.RowsAffected() != 0 {
		tb.Fatalf("%s: user A deleted %d rows of user B", table, tag.RowsAffected())
	}

	// The write side: copy one of A's own rows with B as the owner, the shape
	// of the mistake a bug makes. WITH CHECK must reject it.
	_, err = conn.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %[1]s
		SELECT (jsonb_populate_record(NULL::%[1]s,
			to_jsonb(existing) || jsonb_build_object('user_id', $1::uuid)
				|| CASE WHEN to_jsonb(existing) ? 'id' THEN jsonb_build_object('id', uuidv7()) ELSE '{}' END
				|| CASE WHEN to_jsonb(existing) ? 'key' THEN jsonb_build_object('key', 'copy') ELSE '{}' END)).*
		FROM (SELECT * FROM %[1]s LIMIT 1) existing`, name), b)
	if err == nil {
		tb.Fatalf("%s: user A inserted a row owned by user B", table)
	}
	// Postgres checks the policy before any constraint, so anything other than
	// an RLS violation means the insert failed for a reason that proves nothing.
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		tb.Fatalf("%s: inserting a row owned by user B failed with %v, not the row level security policy", table, err)
	}
}
