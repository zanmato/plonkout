// Package maintenance removes what can never be used again: expired passkey
// ceremonies, sessions, codes and tokens, and abandoned OAuth clients.
package maintenance

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/maintenance/maintenancedb"
	"github.com/zanmato/plonkout/server/internal/platform/schedule"
)

// Prune runs one pass and reports how many rows each step removed.
func Prune(ctx context.Context, pool *pgxpool.Pool) (map[string]int64, error) {
	q := maintenancedb.New(pool)
	steps := []struct {
		name string
		run  func(context.Context) (int64, error)
	}{
		{"ceremonies", q.PruneCeremonies},
		{"sessions", q.PruneSessions},
		{"pow", q.PrunePow},
		{"codes", q.PruneCodes},
		{"access_tokens", q.PruneAccessTokens},
		{"grants", q.PruneGrants},
		{"clients", q.PruneClients},
	}
	removed := make(map[string]int64, len(steps))
	for _, step := range steps {
		n, err := step.run(ctx)
		if err != nil {
			return removed, err
		}
		removed[step.name] = n
	}
	return removed, nil
}

// Job prunes every hour, and once when the server starts.
func Job(pool *pgxpool.Pool, logger *slog.Logger) schedule.Job {
	run := func(ctx context.Context) error {
		removed, err := Prune(ctx, pool)
		if err != nil {
			return err
		}
		logger.Debug("pruned expired rows", "removed", removed)
		return nil
	}
	return schedule.Job{Name: "prune", Spec: "@hourly", Run: run, AtStart: run}
}
