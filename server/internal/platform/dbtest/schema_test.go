package dbtest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zanmato/plonkout/server/internal/platform/dbtest"
)

// userTables are the tables holding a user's data. The coverage test below
// fails when a new public table is not listed here, so it cannot skip the
// isolation test by accident.
var userTables = []string{
	"exercises", "plans", "planned_sessions", "planned_exercises", "planned_targets",
	"workouts", "workout_exercises", "workout_sets", "templates", "settings",
}

// seed creates one row in every user table, acting as the user through the
// application role, so the user_id defaults and the permissive side of every
// policy are exercised too.
func seed(t *testing.T, d *dbtest.DB, userID uuid.UUID) map[string]uuid.UUID {
	t.Helper()
	ctx := dbtest.As(t.Context(), userID)
	ids := map[string]uuid.UUID{}

	insert := func(name, sql string, args ...any) {
		t.Helper()
		var id uuid.UUID
		if err := d.App.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
		ids[name] = id
	}

	insert("exercise", `INSERT INTO exercises (name, muscle_group, single_arm) VALUES ('Side Pressure', 'Shoulders', true) RETURNING id`)
	insert("plan", `INSERT INTO plans (name) VALUES ('9 weeks') RETURNING id`)
	insert("session", `INSERT INTO planned_sessions (plan_id, position, label, week, day, intensity)
		VALUES ($1, 0, 'W1 A', 1, 'A', 'heavy') RETURNING id`, ids["plan"])
	insert("planned_exercise", `INSERT INTO planned_exercises (session_id, position, exercise_id, off_arm_percent)
		VALUES ($1, 0, $2, 82.5) RETURNING id`, ids["session"], ids["exercise"])
	insert("target", `INSERT INTO planned_targets (planned_exercise_id, position, set_type, sets, reps, weight, rpe_min, rpe_max)
		VALUES ($1, 0, 'Top set', 1, 5, 21.25, 8, 9) RETURNING id`, ids["planned_exercise"])
	insert("workout", `INSERT INTO workouts (name, started, planned_session_id) VALUES ('Arms', now(), $1) RETURNING id`, ids["session"])
	insert("workout_exercise", `INSERT INTO workout_exercises (workout_id, position, exercise_id, name, planned_exercise_id)
		VALUES ($1, 0, $2, 'Side Pressure', $3) RETURNING id`, ids["workout"], ids["exercise"], ids["planned_exercise"])
	insert("set", `INSERT INTO workout_sets (workout_exercise_id, position, weight, reps, rpe, arm, target_id, target_seq)
		VALUES ($1, 0, 21, 5, 7, 'right', $2, 1) RETURNING id`, ids["workout_exercise"], ids["target"])
	insert("template", `INSERT INTO templates (name, doc) VALUES ('Arms', '{"exercises":[]}') RETURNING id`)

	if _, err := d.App.Exec(ctx, `INSERT INTO settings (key, value) VALUES ('weightUnit', '"kg"')`); err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	return ids
}

func TestEveryPublicTableHasRowLevelSecurity(t *testing.T) {
	d := dbtest.New(t)

	rows, err := d.Owner.Query(t.Context(), `
		SELECT c.relname, c.relrowsecurity,
			EXISTS (SELECT 1 FROM pg_policies p WHERE p.schemaname = 'public' AND p.tablename = c.relname)
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relname <> '__migrations'
		ORDER BY c.relname`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	listed := map[string]bool{}
	for _, table := range userTables {
		listed[table] = true
	}

	for rows.Next() {
		var name string
		var rls, policy bool
		if err := rows.Scan(&name, &rls, &policy); err != nil {
			t.Fatal(err)
		}
		if !rls || !policy {
			t.Errorf("%s: row level security enabled %v, has a policy %v", name, rls, policy)
		}
		if !listed[name] {
			t.Errorf("%s is a public table missing from userTables, so its isolation is untested", name)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestUsersAreIsolated(t *testing.T) {
	d := dbtest.New(t)
	a, b := d.NewUser(t), d.NewUser(t)
	seed(t, d, a)
	seed(t, d, b)

	for _, table := range userTables {
		t.Run(table, func(t *testing.T) {
			d.AssertIsolated(t, table, a, b)
		})
	}
}

func TestNoUserSeesNothing(t *testing.T) {
	d := dbtest.New(t)
	seed(t, d, d.NewUser(t))

	for _, table := range userTables {
		var count int64
		// A plain context carries no user, so app.user_id is unset.
		err := d.App.QueryRow(context.Background(), `SELECT count(*) FROM `+pgx.Identifier{table}.Sanitize()).Scan(&count)
		if err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s: a connection with no user sees %d rows", table, count)
		}
	}
}

func TestCannotReferenceAnotherUsersRows(t *testing.T) {
	d := dbtest.New(t)
	a, b := d.NewUser(t), d.NewUser(t)
	theirs := seed(t, d, a)
	seed(t, d, b)

	ctx := dbtest.As(t.Context(), b)
	// Positions collide with A's rows on purpose: a unique constraint checked
	// before the foreign key would otherwise answer "duplicate" and confirm
	// the row exists.
	cases := map[string]struct {
		sql  string
		args []any
	}{
		"exercise into A's workout": {
			`INSERT INTO workout_exercises (workout_id, position, name) VALUES ($1, 0, 'x')`,
			[]any{theirs["workout"]},
		},
		"session into A's plan": {
			`INSERT INTO planned_sessions (plan_id, position) VALUES ($1, 0)`,
			[]any{theirs["plan"]},
		},
		"set into A's workout exercise": {
			`INSERT INTO workout_sets (workout_exercise_id, position) VALUES ($1, 0)`,
			[]any{theirs["workout_exercise"]},
		},
		"target into A's planned exercise": {
			`INSERT INTO planned_targets (planned_exercise_id, position) VALUES ($1, 0)`,
			[]any{theirs["planned_exercise"]},
		},
		"planned exercise into A's session": {
			`INSERT INTO planned_exercises (session_id, position, exercise_id) VALUES ($1, 0, $2)`,
			[]any{theirs["session"], theirs["exercise"]},
		},
		"workout linked to A's session": {
			`INSERT INTO workouts (started, planned_session_id) VALUES (now(), $1)`,
			[]any{theirs["session"]},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := d.App.Exec(ctx, tc.sql, tc.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
				t.Fatalf("expected a foreign key violation, got %v", err)
			}
		})
	}
}

func TestDeletingAUserRemovesTheirData(t *testing.T) {
	d := dbtest.New(t)
	a := d.NewUser(t)
	seed(t, d, a)

	if _, err := d.App.Exec(t.Context(), `DELETE FROM auth.users WHERE id = $1`, a); err != nil {
		t.Fatal(err)
	}
	for _, table := range userTables {
		var count int64
		if err := d.Owner.QueryRow(t.Context(), `SELECT count(*) FROM `+pgx.Identifier{table}.Sanitize()).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%s still holds %d rows of a deleted user", table, count)
		}
	}
}
