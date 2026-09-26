-- name: InsertJob :exec
INSERT INTO jobs (kind, payload, max_attempts, lease_seconds, sensitive)
VALUES ($1, $2, $3, $4, $5);

-- name: ClaimJobs :many
-- Claims up to max_jobs runnable jobs: pending ones that are due, and running
-- ones whose lease expired (their worker died). SKIP LOCKED lets concurrent
-- workers claim different jobs instead of waiting on each other.
UPDATE jobs
SET status      = 'running',
    attempts    = attempts + 1,
    lease_until = now() + make_interval(secs => lease_seconds),
    updated_at  = now()
WHERE id IN (
    SELECT id FROM jobs
    WHERE attempts < max_attempts
      AND ((status = 'pending' AND run_after <= now())
        OR (status = 'running' AND lease_until < now()))
    ORDER BY run_after, id
    LIMIT sqlc.arg(max_jobs)
    FOR UPDATE SKIP LOCKED
)
RETURNING id, kind, payload, attempts, max_attempts, lease_seconds;

-- name: CompleteJob :execrows
-- Only the attempt that still holds the job may record its outcome.
DELETE FROM jobs
WHERE id = $1 AND status = 'running' AND attempts = $2;

-- name: RetryJob :execrows
UPDATE jobs
SET status      = 'pending',
    lease_until = NULL,
    run_after   = now() + make_interval(secs => sqlc.arg(delay_seconds)::float8),
    last_error  = sqlc.arg(last_error),
    updated_at  = now()
WHERE id = sqlc.arg(id) AND status = 'running' AND attempts = sqlc.arg(attempts);

-- name: FailJob :execrows
UPDATE jobs
SET status      = 'failed',
    lease_until = NULL,
    failed_at   = now(),
    last_error  = sqlc.arg(last_error),
    payload     = CASE WHEN sensitive THEN NULL ELSE payload END,
    updated_at  = now()
WHERE id = sqlc.arg(id) AND status = 'running' AND attempts = sqlc.arg(attempts);

-- name: FailAbandonedJobs :execrows
-- Final attempts whose worker vanished: the lease expired and no attempts remain.
UPDATE jobs
SET status      = 'failed',
    lease_until = NULL,
    failed_at   = now(),
    last_error  = 'lease expired during the final attempt',
    payload     = CASE WHEN sensitive THEN NULL ELSE payload END,
    updated_at  = now()
WHERE status = 'running' AND lease_until < now() AND attempts >= max_attempts;

-- name: DeleteExpiredFailedJobs :execrows
DELETE FROM jobs
WHERE status = 'failed' AND failed_at < now() - interval '14 days';

-- name: JobCounts :many
SELECT kind, status, count(*)::integer AS jobs
FROM jobs
GROUP BY kind, status
ORDER BY kind, status;

-- name: OldestOverdueJobSeconds :one
-- How late the most overdue runnable job is, in seconds; 0 when nothing waits.
SELECT coalesce(extract(epoch FROM now() - min(run_after)), 0)::float8 AS overdue_seconds
FROM jobs
WHERE status = 'pending' AND run_after <= now();

-- name: RecentFailedJobs :many
SELECT id, kind, attempts, last_error, failed_at
FROM jobs
WHERE status = 'failed'
ORDER BY failed_at DESC
LIMIT 10;
