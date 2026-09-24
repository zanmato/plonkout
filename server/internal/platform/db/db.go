// Package db opens the connection pool the server runs on.
//
// There is one constructor, and it always installs the user hook. A pool built
// any other way would still run queries and see no rows at all, or with the
// owner role every user's rows, and neither is something a caller should be
// able to get by accident.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/platform/userctx"
)

// Open builds the pool and checks the database is reachable.
func Open(ctx context.Context, dsn string, maxConns int32, logger *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := build(dsn, maxConns, logger)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("reach the database: %w", err)
	}
	return pool, nil
}

// OpenWithoutConnecting builds the same pool as Open and never contacts the
// database. The openapi subcommand builds the real server to print its
// document and must not need a running Postgres to do it. pgxpool connects
// lazily, so the pool simply fails on its first query.
func OpenWithoutConnecting(dsn string, logger *slog.Logger) (*pgxpool.Pool, error) {
	return build(dsn, 1, logger)
}

func build(dsn string, maxConns int32, logger *slog.Logger) (*pgxpool.Pool, error) {
	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	if maxConns > 0 {
		conf.MaxConns = maxConns
	}
	conf.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}
	installUserHook(conf, logger)

	pool, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	return pool, nil
}

// installUserHook wires the user from the context into the connection.
//
// PrepareConn runs before a connection leaves the pool and AfterRelease runs
// when it comes back, which makes the user a property of the checkout rather
// than of a query. Clearing on release is the half that matters most: pooled
// connections are reused, and a leftover app.user_id would hand the next
// borrower somebody else's rows.
func installUserHook(conf *pgxpool.Config, logger *slog.Logger) {
	conf.PrepareConn = func(ctx context.Context, conn *pgx.Conn) (bool, error) {
		userID, ok := userctx.From(ctx)
		if !ok {
			// No user means no rows, not every row. The policies compare
			// against NULL and filter everything out.
			return true, nil
		}
		if _, err := conn.Exec(ctx, "SELECT set_config('app.user_id', $1, false)", userID.String()); err != nil {
			logger.Error("failed to set the user on a connection", "error", err)
			return false, err
		}
		return true, nil
	}

	conf.AfterRelease = func(conn *pgx.Conn) bool {
		// The request context is gone by now. A failed reset drops the
		// connection rather than returning it dirty.
		if _, err := conn.Exec(context.Background(), "SELECT set_config('app.user_id', '', false)"); err != nil {
			logger.Error("failed to clear the user on a connection", "error", err)
			return false
		}
		return true
	}
}

// TxBeginner is what RunInTx needs. A pool and a transaction both satisfy it.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// RunInTx runs fn in a transaction, committing when it returns nil and rolling
// back otherwise.
func RunInTx(ctx context.Context, beginner TxBeginner, fn func(tx pgx.Tx) error) error {
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rollbackErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
