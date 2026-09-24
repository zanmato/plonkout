// Package maintenance removes what can never be used again: expired passkey
// ceremonies, sessions, codes and tokens, and abandoned OAuth clients.
package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/maintenance/maintenancedb"
)

// Interval is how often the pruner runs.
const Interval = time.Hour

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

// Run prunes now and then every Interval until the context ends. One server
// instance is all there is, so no lock is needed to keep two from racing.
func Run(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	for {
		removed, err := Prune(ctx, pool)
		if err != nil && ctx.Err() == nil {
			logger.Error("pruning failed", "error", err)
		} else if err == nil {
			logger.Debug("pruned expired rows", "removed", removed)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
