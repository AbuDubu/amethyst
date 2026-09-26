// Command amethyst runs one Amethyst server.
//
// Usage:
//
//	amethyst migrate   apply pending database migrations, then exit
//	amethyst serve     run the HTTP server and background job worker
//	amethyst jobs      show background job queue status
//	amethyst version   print the build version
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
	"text/tabwriter"
	"time"

	"github.com/AbuDubu/amethyst/server/internal/api"
	"github.com/AbuDubu/amethyst/server/internal/federation"
	"github.com/AbuDubu/amethyst/server/internal/federation/store"
	"github.com/AbuDubu/amethyst/server/internal/platform/config"
	"github.com/AbuDubu/amethyst/server/internal/platform/db"
	"github.com/AbuDubu/amethyst/server/internal/platform/httpserver"
	"github.com/AbuDubu/amethyst/server/internal/platform/jobs"
)

const usage = "usage: amethyst <migrate|serve|jobs|version>"

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
	case "jobs":
		err = showJobs(ctx)
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

	// Job kinds are registered here as features add them (email in #11).
	registry := jobs.NewRegistry()
	worker := jobs.NewWorker(pool, registry, logger)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker.Run(ctx)
	}()

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
	// The worker stopped claiming when ctx was cancelled; wait for running jobs.
	<-workerDone
	return nil
}

func showJobs(ctx context.Context) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	stats, err := jobs.ReadStats(ctx, pool)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tSTATUS\tJOBS")
	for _, c := range stats.Counts {
		fmt.Fprintf(w, "%s\t%s\t%d\n", c.Kind, c.Status, c.Jobs)
	}
	if len(stats.Counts) == 0 {
		fmt.Fprintln(w, "(queue empty)\t\t")
	}
	_ = w.Flush()
	fmt.Printf("\nMost overdue runnable job: %s\n", stats.Overdue.Round(time.Second))
	if len(stats.RecentFailures) > 0 {
		fmt.Println("\nRecent failures:")
		for _, f := range stats.RecentFailures {
			fmt.Printf("  %s  %s  %s after %d attempts: %s\n",
				f.FailedAt.Format(time.RFC3339), f.ID, f.Kind, f.Attempts, *f.LastError)
		}
	}
	return nil
}
