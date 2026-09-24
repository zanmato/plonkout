package maintenance_test

import (
	"testing"

	"github.com/zanmato/plonkout/server/internal/maintenance"
	"github.com/zanmato/plonkout/server/internal/platform/dbtest"
)

func TestPruneRemovesOnlyWhatExpired(t *testing.T) {
	d := dbtest.New(t)
	ctx := t.Context()
	user := d.NewUser(t)

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := d.App.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO auth.web_sessions (token_hash, user_id, expires_at, absolute_expires_at) VALUES
		('\x01', $1, now() - interval '1 minute', now() + interval '1 day'),
		('\x02', $1, now() + interval '1 day', now() + interval '2 days')`, user)
	exec(`INSERT INTO auth.webauthn_ceremonies (kind, session_data, expires_at) VALUES
		('login', '{}', now() - interval '1 minute'), ('login', '{}', now() + interval '5 minutes')`)
	exec(`INSERT INTO auth.oauth_clients (client_id, kind, redirect_uris, created_at) VALUES
		('old_unused', 'dcr', '{https://a/cb}', now() - interval '8 days'),
		('old_used', 'dcr', '{https://a/cb}', now() - interval '8 days'),
		('new_unused', 'dcr', '{https://a/cb}', now())`)
	exec(`INSERT INTO auth.oauth_grants (client_id, user_id, scope, resource, refresh_expires_at)
		VALUES ('old_used', $1, 'training', 'x', now() + interval '1 day')`, user)

	removed, err := maintenance.Prune(ctx, d.App)
	if err != nil {
		t.Fatal(err)
	}
	if removed["sessions"] != 1 || removed["ceremonies"] != 1 || removed["clients"] != 1 {
		t.Fatalf("unexpected pruning %v", removed)
	}

	var clients []string
	rows, err := d.App.Query(ctx, `SELECT client_id FROM auth.oauth_clients ORDER BY client_id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		clients = append(clients, id)
	}
	if len(clients) != 2 || clients[0] != "new_unused" || clients[1] != "old_used" {
		t.Fatalf("unexpected clients left %v", clients)
	}
}
