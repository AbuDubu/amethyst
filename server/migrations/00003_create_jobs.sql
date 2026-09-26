-- +goose Up

-- Durable background work. A job is inserted in the same transaction as the
-- change that requires it, so it exists if and only if that change committed.
-- Succeeded jobs are deleted; failed ones are kept briefly for inspection.
CREATE TABLE jobs (
    id            uuid        PRIMARY KEY DEFAULT uuidv7(),
    kind          text        NOT NULL CHECK (kind ~ '^[a-z][a-z0-9_.]*$'),
    -- NULL once scrubbed: sensitive payloads are removed when a job fails.
    payload       jsonb,
    status        text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'failed')),
    -- Incremented when a worker claims the job, so a job that crashes its
    -- worker still uses up attempts instead of retrying forever. Also serves as
    -- a fencing token: outcomes are recorded only for the attempt that is
    -- still current.
    attempts      integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    max_attempts  integer     NOT NULL CHECK (max_attempts > 0),
    lease_seconds integer     NOT NULL CHECK (lease_seconds > 0),
    sensitive     boolean     NOT NULL DEFAULT false,
    run_after     timestamptz NOT NULL DEFAULT now(),
    lease_until   timestamptz,
    last_error    text,
    failed_at     timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'running') = (lease_until IS NOT NULL)),
    CHECK ((status = 'failed') = (failed_at IS NOT NULL))
);

-- Finding work: ready pending jobs, and running jobs whose lease expired.
CREATE INDEX jobs_pending_ready ON jobs (run_after) WHERE status = 'pending';
CREATE INDEX jobs_running_lease ON jobs (lease_until) WHERE status = 'running';
CREATE INDEX jobs_failed_at ON jobs (failed_at) WHERE status = 'failed';

-- +goose Down
DROP TABLE jobs;
