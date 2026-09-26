package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AbuDubu/amethyst/server/internal/platform/jobs/store"
)

const (
	defaultConcurrency   = 4
	defaultPollInterval  = time.Second
	defaultShutdownGrace = 10 * time.Second
	maintenanceInterval  = time.Minute
	outcomeTimeout       = 5 * time.Second
	maxErrorLength       = 1000
)

// Worker runs jobs from the queue.
type Worker struct {
	pool     *pgxpool.Pool
	registry *Registry
	logger   *slog.Logger

	// Concurrency is how many jobs run at once.
	Concurrency int
	// PollInterval is how long to wait before looking again when idle.
	PollInterval time.Duration
	// ShutdownGrace is how long running jobs may continue after Run's context
	// is cancelled before their contexts are cancelled too.
	ShutdownGrace time.Duration
}

func NewWorker(pool *pgxpool.Pool, registry *Registry, logger *slog.Logger) *Worker {
	return &Worker{
		pool:          pool,
		registry:      registry,
		logger:        logger,
		Concurrency:   defaultConcurrency,
		PollInterval:  defaultPollInterval,
		ShutdownGrace: defaultShutdownGrace,
	}
}

// Run processes jobs until ctx is cancelled, then waits for running jobs to
// finish (cancelling them after ShutdownGrace).
func (w *Worker) Run(ctx context.Context) {
	// Handlers get their own context so that stopping Run does not abort work
	// already in progress until the grace period is over.
	jobCtx, cancelJobs := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelJobs()
	stopGrace := context.AfterFunc(ctx, func() { time.AfterFunc(w.ShutdownGrace, cancelJobs) })
	defer stopGrace()

	var running sync.WaitGroup
	defer running.Wait()
	slots := make(chan struct{}, w.Concurrency)
	nextMaintenance := time.Now()

	for ctx.Err() == nil {
		if time.Now().After(nextMaintenance) {
			w.Maintain(ctx)
			nextMaintenance = time.Now().Add(maintenanceInterval)
		}

		free := w.Concurrency - len(slots)
		claimed, err := w.claim(ctx, free)
		if err != nil && ctx.Err() == nil {
			w.logger.Error("claim jobs", "error", err)
		}
		for _, j := range claimed {
			slots <- struct{}{}
			running.Go(func() {
				defer func() { <-slots }()
				w.run(jobCtx, j)
			})
		}

		// A full batch suggests more work is waiting; otherwise, pause.
		if len(claimed) > 0 && len(claimed) == free {
			continue
		}
		select {
		case <-ctx.Done():
		case <-time.After(w.PollInterval):
		}
	}
}

// RunOnce claims up to Concurrency due jobs and runs them to completion,
// returning how many ran. Tests use it to process the queue deterministically.
func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	claimed, err := w.claim(ctx, w.Concurrency)
	if err != nil {
		return 0, err
	}
	var wg sync.WaitGroup
	for _, j := range claimed {
		wg.Go(func() { w.run(ctx, j) })
	}
	wg.Wait()
	return len(claimed), nil
}

// Maintain fails jobs abandoned during their final attempt and deletes failed
// jobs older than the retention period. Run calls it periodically.
func (w *Worker) Maintain(ctx context.Context) {
	q := store.New(w.pool)
	if n, err := q.FailAbandonedJobs(ctx); err != nil {
		w.logger.Error("fail abandoned jobs", "error", err)
	} else if n > 0 {
		w.logger.Warn("jobs failed after their final attempt was abandoned", "count", n)
	}
	if _, err := q.DeleteExpiredFailedJobs(ctx); err != nil {
		w.logger.Error("delete expired failed jobs", "error", err)
	}
}

func (w *Worker) claim(ctx context.Context, n int) ([]store.ClaimJobsRow, error) {
	if n <= 0 {
		return nil, nil
	}
	return store.New(w.pool).ClaimJobs(ctx, int32(n))
}

// run executes one claimed job and records its outcome.
func (w *Worker) run(ctx context.Context, j store.ClaimJobsRow) {
	log := w.logger.With("job_id", j.ID, "kind", j.Kind, "attempt", j.Attempts)
	err := w.execute(ctx, j)

	// Record the outcome even if ctx was cancelled during shutdown.
	octx, cancel := context.WithTimeout(context.WithoutCancel(ctx), outcomeTimeout)
	defer cancel()
	q := store.New(w.pool)

	var rows int64
	var outcomeErr error
	switch {
	case err == nil:
		rows, outcomeErr = q.CompleteJob(octx, store.CompleteJobParams{ID: j.ID, Attempts: j.Attempts})
	case j.Attempts >= j.MaxAttempts:
		log.Error("job failed permanently", "error", err)
		rows, outcomeErr = q.FailJob(octx, store.FailJobParams{ID: j.ID, Attempts: j.Attempts, LastError: new(truncate(err.Error()))})
	default:
		delay := Backoff(int(j.Attempts), rand.Float64)
		log.Warn("job failed; will retry", "error", err, "retry_in", delay.Round(time.Second))
		rows, outcomeErr = q.RetryJob(octx, store.RetryJobParams{
			ID: j.ID, Attempts: j.Attempts, DelaySeconds: delay.Seconds(), LastError: new(truncate(err.Error())),
		})
	}
	switch {
	case outcomeErr != nil:
		// The lease will expire and the job will run again: at-least-once.
		log.Error("record job outcome", "error", outcomeErr)
	case rows == 0:
		// Our lease expired and another attempt took over; its outcome wins.
		log.Warn("job outcome discarded: lease was lost to a newer attempt")
	}
}

// execute runs the handler within the lease, turning panics into errors so
// one bad job cannot take down the server.
func (w *Worker) execute(ctx context.Context, j store.ClaimJobsRow) (err error) {
	handler, ok := w.registry.handlers[j.Kind]
	if !ok {
		return fmt.Errorf("no handler registered for kind %q", j.Kind)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(j.LeaseSeconds)*time.Second)
	defer cancel()
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("handler panicked: %v", p)
		}
	}()
	if err := handler(ctx, j.Payload); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("exceeded its %ds lease: %w", j.LeaseSeconds, err)
		}
		return err
	}
	return nil
}

func truncate(s string) string {
	if len(s) > maxErrorLength {
		return s[:maxErrorLength] + "…"
	}
	return s
}
