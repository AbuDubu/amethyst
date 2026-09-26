package jobs

import (
	"context"
	"time"

	"github.com/AbuDubu/amethyst/server/internal/platform/jobs/store"
)

// Stats summarizes the queue for operators: how much work is waiting or
// failed, and whether anything is stuck.
type Stats struct {
	Counts []store.JobCountsRow
	// Overdue is how late the most overdue runnable job is. A value that keeps
	// growing means workers are not keeping up or not running.
	Overdue        time.Duration
	RecentFailures []store.RecentFailedJobsRow
}

func ReadStats(ctx context.Context, db store.DBTX) (Stats, error) {
	q := store.New(db)
	counts, err := q.JobCounts(ctx)
	if err != nil {
		return Stats{}, err
	}
	overdue, err := q.OldestOverdueJobSeconds(ctx)
	if err != nil {
		return Stats{}, err
	}
	failures, err := q.RecentFailedJobs(ctx)
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		Counts:         counts,
		Overdue:        time.Duration(overdue * float64(time.Second)),
		RecentFailures: failures,
	}, nil
}
