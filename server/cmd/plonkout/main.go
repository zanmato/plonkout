// Command plonkout is the server.
//
//	plonkout [-config path] [-m | -migrate-only]   serve, optionally migrating first
//	plonkout openapi                               print the OpenAPI document
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zanmato/plonkout/server/internal/maintenance"
	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/platform/migrate"
	"github.com/zanmato/plonkout/server/internal/server"
)

func main() {
	configPath := flag.String("config", "", "path to the config file (default $PLONKOUT_CONFIG or ./config.toml)")
	migrateFirst := flag.Bool("m", false, "run migrations, then serve")
	migrateOnly := flag.Bool("migrate-only", false, "run migrations and exit")
	flag.Parse()

	if flag.Arg(0) == "openapi" {
		if err := printOpenAPI(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := run(*configPath, *migrateFirst, *migrateOnly); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(configPath string, migrateFirst, migrateOnly bool) error {
	cfg, err := config.Load(config.Path(configPath))
	if err != nil {
		return err
	}
	logger := cfg.Logger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if migrateFirst || migrateOnly {
		dir := migrate.Dir(cfg.Database.MigrationsPath)
		logger.Info("running migrations", "dir", dir)
		if err := migrate.Up(ctx, cfg.Database.Migrate, dir, logger); err != nil {
			return err
		}
		if migrateOnly {
			return nil
		}
	}

	if !cfg.RLSEnforced() {
		logger.Warn("database.app and database.migrate are the same connection, row level security is not enforced")
	}

	pool, err := db.Open(ctx, cfg.Database.App, cfg.Database.MaxConns, logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	srv, err := server.New(server.Deps{Config: cfg, Pool: pool, Logger: logger})
	if err != nil {
		return err
	}
	go maintenance.Run(ctx, pool, logger)

	httpServer := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Server.Addr, "base_url", cfg.Server.BaseURL)
		errs <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// printOpenAPI builds the real server from placeholder configuration and never
// contacts a database, so generating the committed document works in a fresh
// checkout.
func printOpenAPI() error {
	cfg := &config.Config{
		Server:   config.Server{BaseURL: "http://localhost"},
		WebAuthn: config.WebAuthn{RPID: "localhost", RPName: "Plonkout", Origins: []string{"http://localhost"}},
	}
	logger := slog.New(slog.DiscardHandler)

	pool, err := db.OpenWithoutConnecting("postgres://localhost/plonkout", logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	srv, err := server.New(server.Deps{Config: cfg, Pool: pool, Logger: logger})
	if err != nil {
		return err
	}
	doc, err := srv.OpenAPI()
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(doc)
	return err
}
