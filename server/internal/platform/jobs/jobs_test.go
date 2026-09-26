package jobs_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
	"github.com/AbuDubu/amethyst/server/internal/platform/jobs"
	"github.com/AbuDubu/amethyst/server/internal/platform/jobs/store"
)

type greeting struct {
	Name string `json:"name"`
}

var greet = jobs.Kind[greeting]{Name: "test.greet"}

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func newWorker(pool *pgxpool.Pool, r *jobs.Registry) *jobs.Worker {
	return jobs.NewWorker(pool, r, discard)
}

type jobRow struct {
	Status    string
	Attempts  int
	Payload   []byte
	LastError *string
	RunAfter  time.Time
}

// onlyJob returns the single job in the table.
func onlyJob(t *testing.T, pool *pgxpool.Pool) jobRow {
	t.Helper()
	var j jobRow
	err := pool.QueryRow(context.Background(),
		`SELECT status, attempts, payload, last_error, run_after FROM jobs`).
		Scan(&j.Status, &j.Attempts, &j.Payload, &j.LastError, &j.RunAfter)
	if err != nil {
		t.Fatalf("load the only job: %v", err)
	}
	return j
}

func jobCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM jobs`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEnqueueCommitsAndRollsBackWithTheTransaction(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	tx, _ := pool.Begin(ctx)
	if err := greet.Enqueue(ctx, tx, greeting{Name: "rolled back"}); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(ctx)
	if n := jobCount(t, pool); n != 0 {
		t.Fatalf("jobs after rollback = %d, want 0", n)
	}

	tx, _ = pool.Begin(ctx)
	if err := greet.Enqueue(ctx, tx, greeting{Name: "committed"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if n := jobCount(t, pool); n != 1 {
		t.Fatalf("jobs after commit = %d, want 1", n)
	}
}

func TestSuccessfulJobRunsWithItsPayloadAndIsDeleted(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	r := jobs.NewRegistry()
	var got greeting
	jobs.Handle(r, greet, func(_ context.Context, g greeting) error { got = g; return nil })

	if err := greet.Enqueue(ctx, pool, greeting{Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	n, err := newWorker(pool, r).RunOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("RunOnce = %d, %v; want 1 job", n, err)
	}
	if got.Name != "alice" {
		t.Errorf("handler received %+v, want alice", got)
	}
	if n := jobCount(t, pool); n != 0 {
		t.Errorf("jobs after success = %d, want 0", n)
	}
}

func TestFailedJobIsRetriedLaterWithBackoff(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	r := jobs.NewRegistry()
	jobs.Handle(r, greet, func(context.Context, greeting) error { return errors.New("smtp unavailable") })
	w := newWorker(pool, r)

	_ = greet.Enqueue(ctx, pool, greeting{Name: "alice"})
	start := time.Now()
	if _, err := w.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}

	j := onlyJob(t, pool)
	if j.Status != "pending" || j.Attempts != 1 || j.LastError == nil || *j.LastError != "smtp unavailable" {
		t.Fatalf("after one failure: %+v, want pending, 1 attempt, error recorded", j)
	}
	// First retry: one minute ±25%.
	if wait := j.RunAfter.Sub(start); wait < 44*time.Second || wait > 76*time.Second {
		t.Errorf("retry scheduled after %v, want about 1m", wait)
	}
	// Not due yet, so an immediate second pass finds nothing.
	if n, _ := w.RunOnce(ctx); n != 0 {
		t.Errorf("RunOnce before the retry is due ran %d jobs, want 0", n)
	}
}

func TestFinalFailureMarksFailedAndScrubsSensitivePayloads(t *testing.T) {
	for _, sensitive := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "sensitive"}[sensitive], func(t *testing.T) {
			ctx := context.Background()
			pool := dbtest.New(t)
			kind := jobs.Kind[greeting]{Name: "test.once", Options: jobs.Options{MaxAttempts: 1, Sensitive: sensitive}}
			r := jobs.NewRegistry()
			jobs.Handle(r, kind, func(context.Context, greeting) error { return errors.New("boom") })

			_ = kind.Enqueue(ctx, pool, greeting{Name: "secret-link"})
			if _, err := newWorker(pool, r).RunOnce(ctx); err != nil {
				t.Fatal(err)
			}

			j := onlyJob(t, pool)
			if j.Status != "failed" {
				t.Fatalf("status = %s, want failed", j.Status)
			}
			if scrubbed := j.Payload == nil; scrubbed != sensitive {
				t.Errorf("payload scrubbed = %v, want %v", scrubbed, sensitive)
			}
		})
	}
}

func TestExpiredLeaseIsReclaimedAndTheStaleOutcomeIsDiscarded(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	q := store.New(pool)
	_ = greet.Enqueue(ctx, pool, greeting{Name: "alice"})

	// Worker 1 claims the job, then stalls past its lease.
	first, err := q.ClaimJobs(ctx, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim = %v, %v", first, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET lease_until = now() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}

	// Worker 2 reclaims it as a new attempt.
	second, err := q.ClaimJobs(ctx, 1)
	if err != nil || len(second) != 1 || second[0].Attempts != 2 {
		t.Fatalf("reclaim = %+v, %v; want the same job at attempt 2", second, err)
	}

	// Worker 1 wakes up and reports success for attempt 1: ignored.
	rows, err := q.CompleteJob(ctx, store.CompleteJobParams{ID: first[0].ID, Attempts: first[0].Attempts})
	if err != nil || rows != 0 {
		t.Errorf("stale completion affected %d rows (err %v), want 0", rows, err)
	}
	if j := onlyJob(t, pool); j.Status != "running" || j.Attempts != 2 {
		t.Errorf("job = %+v, want still running attempt 2", j)
	}
}

func TestConcurrentWorkersRunEachJobOnce(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	const total = 60

	var mu sync.Mutex
	runs := map[string]int{}
	r := jobs.NewRegistry()
	jobs.Handle(r, greet, func(_ context.Context, g greeting) error {
		mu.Lock()
		runs[g.Name]++
		mu.Unlock()
		time.Sleep(5 * time.Millisecond) // widen the window for overlap
		return nil
	})
	for i := range total {
		_ = greet.Enqueue(ctx, pool, greeting{Name: string(rune('A' + i))})
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			w := newWorker(pool, r)
			for {
				if n, err := w.RunOnce(ctx); err != nil || n == 0 {
					return
				}
			}
		})
	}
	wg.Wait()

	if len(runs) != total {
		t.Errorf("%d distinct jobs ran, want %d", len(runs), total)
	}
	for name, n := range runs {
		if n != 1 {
			t.Errorf("job %q ran %d times, want 1", name, n)
		}
	}
}

func TestUnknownKindAndPanicsAreFailuresNotCrashes(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	r := jobs.NewRegistry()
	panicky := jobs.Kind[greeting]{Name: "test.panic"}
	jobs.Handle(r, panicky, func(context.Context, greeting) error { panic("nil map") })
	unknown := jobs.Kind[greeting]{Name: "test.unregistered"}

	_ = panicky.Enqueue(ctx, pool, greeting{})
	_ = unknown.Enqueue(ctx, pool, greeting{})
	if _, err := newWorker(pool, r).RunOnce(ctx); err != nil {
		t.Fatal(err)
	}

	rows, _ := pool.Query(ctx, `SELECT kind, status, last_error FROM jobs ORDER BY kind`)
	defer rows.Close()
	for rows.Next() {
		var kind, status, lastError string
		_ = rows.Scan(&kind, &status, &lastError)
		if status != "pending" || lastError == "" {
			t.Errorf("%s: status %s, error %q; want pending retry with an error", kind, status, lastError)
		}
	}
}

func TestHandlerContextEndsAtTheLease(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	slow := jobs.Kind[greeting]{Name: "test.slow", Options: jobs.Options{Lease: time.Second}}
	r := jobs.NewRegistry()
	jobs.Handle(r, slow, func(ctx context.Context, _ greeting) error {
		<-ctx.Done() // a handler that respects cancellation
		return ctx.Err()
	})

	_ = slow.Enqueue(ctx, pool, greeting{})
	start := time.Now()
	if _, err := newWorker(pool, r).RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("handler ran %v; its context should end at the 1s lease", elapsed)
	}
	if j := onlyJob(t, pool); j.Status != "pending" {
		t.Errorf("status = %s, want pending retry after exceeding the lease", j.Status)
	}
}

func TestMaintenanceFailsAbandonedFinalAttemptsAndExpiresOldFailures(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	once := jobs.Kind[greeting]{Name: "test.once", Options: jobs.Options{MaxAttempts: 1, Sensitive: true}}
	_ = once.Enqueue(ctx, pool, greeting{Name: "secret"})

	// A worker claims the final attempt and dies: the lease expires unrecorded.
	if _, err := store.New(pool).ClaimJobs(ctx, 1); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `UPDATE jobs SET lease_until = now() - interval '1 second'`)

	w := newWorker(pool, jobs.NewRegistry())
	w.Maintain(ctx)
	if j := onlyJob(t, pool); j.Status != "failed" || j.Payload != nil {
		t.Fatalf("abandoned final attempt = %+v, want failed with payload scrubbed", j)
	}

	// Failed jobs are kept for 14 days, then removed.
	_, _ = pool.Exec(ctx, `UPDATE jobs SET failed_at = now() - interval '13 days'`)
	w.Maintain(ctx)
	if n := jobCount(t, pool); n != 1 {
		t.Fatalf("13-day-old failure removed; want it kept")
	}
	_, _ = pool.Exec(ctx, `UPDATE jobs SET failed_at = now() - interval '15 days'`)
	w.Maintain(ctx)
	if n := jobCount(t, pool); n != 0 {
		t.Errorf("15-day-old failure kept; want it deleted")
	}
}

func TestRunProcessesJobsAndStopsGracefully(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := dbtest.New(t)
	var done atomic.Int32
	r := jobs.NewRegistry()
	jobs.Handle(r, greet, func(context.Context, greeting) error { done.Add(1); return nil })
	w := newWorker(pool, r)
	w.PollInterval = 20 * time.Millisecond

	stopped := make(chan struct{})
	go func() { w.Run(ctx); close(stopped) }()

	for range 3 {
		_ = greet.Enqueue(context.Background(), pool, greeting{})
	}
	deadline := time.Now().Add(5 * time.Second)
	for done.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if done.Load() != 3 {
		t.Fatalf("processed %d of 3 jobs", done.Load())
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}

func TestBackoff(t *testing.T) {
	mid := func() float64 { return 0.5 } // no jitter
	for attempt, want := range map[int]time.Duration{
		1: time.Minute, 2: 2 * time.Minute, 3: 4 * time.Minute, 6: 32 * time.Minute, 7: time.Hour, 9: time.Hour,
	} {
		if got := jobs.Backoff(attempt, mid); got != want {
			t.Errorf("Backoff(%d) = %v, want %v", attempt, got, want)
		}
	}
	if lo, hi := jobs.Backoff(1, func() float64 { return 0 }), jobs.Backoff(1, func() float64 { return 0.999 }); lo < 45*time.Second || hi > 75*time.Second {
		t.Errorf("jitter range = %v..%v, want within ±25%% of 1m", lo, hi)
	}

	var total time.Duration
	for attempt := 1; attempt < jobs.DefaultMaxAttempts; attempt++ {
		total += jobs.Backoff(attempt, mid)
	}
	if total < 3*time.Hour || total > 5*time.Hour {
		t.Errorf("default retries span %v, want about 4h", total)
	}
}
