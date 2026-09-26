// Command amethyst runs one Amethyst server.
//
// Usage:
//
//	amethyst migrate   apply pending database migrations, then exit
//	amethyst serve     run the HTTP server
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AbuDubu/amethyst/server/internal/api"
	"github.com/AbuDubu/amethyst/server/internal/federation"
	"github.com/AbuDubu/amethyst/server/internal/federation/store"
	"github.com/AbuDubu/amethyst/server/internal/platform/config"
	"github.com/AbuDubu/amethyst/server/internal/platform/db"
	"github.com/AbuDubu/amethyst/server/internal/platform/httpserver"
)

const usage = "usage: amethyst <migrate|serve|version>"

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	// Stop on Ctrl-C or a container stop signal.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "migrate":
		err = migrate(ctx, logger)
	case "serve":
		err = serve(ctx, logger)
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		logger.Error(os.Args[1]+" failed", "error", err)
		os.Exit(1)
	}
}

// migrate is deliberately a separate step from serve: schema changes happen
// when an operator chooses, never as a side effect of a restart.
func migrate(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	migrator, err := db.NewMigrator(pool)
	if err != nil {
		return err
	}
	applied, err := migrator.Up(ctx)
	if err != nil {
		return err
	}
	logger.Info("migrations complete", "applied", applied)
	return nil
}

func serve(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	migrator, err := db.NewMigrator(pool)
	if err != nil {
		return err
	}
	// Recording this server's identity needs the schema, so pending migrations
	// stop startup here with a clear message.
	if err := migrator.CheckCurrent(ctx); err != nil {
		return err
	}
	local, err := federation.EnsureLocalServer(ctx, store.New(pool), cfg.CanonicalOrigin)
	if err != nil {
		return err
	}
	logger.Info("local server identity", "canonical_origin", local.CanonicalOrigin, "id", local.ID)

	apiHandler, err := api.NewHandler(api.Deps{
		Ready:           db.Ready(pool, migrator),
		CanonicalOrigin: cfg.CanonicalOrigin,
		Version:         version,
		Logger:          logger,
	})
	if err != nil {
		return err
	}
	handler, err := httpserver.New(os.DirFS(cfg.WebDir), apiHandler)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "web_dir", cfg.WebDir, "version", version)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// Give in-flight requests a bounded time to finish.
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
