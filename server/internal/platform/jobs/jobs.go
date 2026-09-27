// Package jobs is a durable background-job queue stored in PostgreSQL.
//
// Enqueue a job in the same transaction as the change that requires it: the
// job then exists if and only if that change committed (a transactional
// outbox). A Worker claims due jobs with a time-limited lease, runs them
// outside any transaction, and records the outcome.
//
// Delivery is at least once. A worker can crash after a handler's side effect
// (an email sent) but before recording success; the lease then expires and the
// job runs again. Handlers must therefore be idempotent or tolerate repeats.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/AbuDubu/amethyst/server/internal/platform/jobs/store"
)

// DB is what Enqueue writes through: a pgx transaction (normally) or pool.
type DB = store.DBTX

// Defaults for Options fields left zero.
const (
	DefaultMaxAttempts = 10
	DefaultLease       = 2 * time.Minute
)

// Retry schedule: the wait doubles from baseDelay up to maxDelay, with ±25%
// jitter so jobs that failed together do not all retry at the same instant.
// With the defaults, retries span roughly four hours before a job fails.
const (
	baseDelay = time.Minute
	maxDelay  = time.Hour
)

// Options configure every job of a kind. They are stored on each job at
// enqueue time.
type Options struct {
	// MaxAttempts is how many times the job may run, including the first.
	MaxAttempts int
	// Lease is how long one attempt may hold the job. The handler's context is
	// cancelled at the lease deadline; after it, another worker may take over.
	Lease time.Duration
	// Sensitive jobs have their payload deleted if they fail permanently, so
	// secrets (such as a verification link) do not linger in the database.
	Sensitive bool
}

func (o Options) withDefaults() Options {
	if o.MaxAttempts == 0 {
		o.MaxAttempts = DefaultMaxAttempts
	}
	if o.Lease == 0 {
		o.Lease = DefaultLease
	}
	return o
}

// Kind declares a type of job whose payload is a T, encoded as JSON. Declare
// kinds as package-level variables in the feature that owns them.
type Kind[T any] struct {
	Name string
	Options
}

// Enqueue adds a job. Pass the transaction that makes the corresponding
// change, so the job commits or rolls back with it.
func (k Kind[T]) Enqueue(ctx context.Context, db DB, payload T) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s payload: %w", k.Name, err)
	}
	o := k.withDefaults()
	return store.New(db).InsertJob(ctx, store.InsertJobParams{
		Kind:         k.Name,
		Payload:      body,
		MaxAttempts:  int32(o.MaxAttempts),
		LeaseSeconds: int32(o.Lease / time.Second),
		Sensitive:    o.Sensitive,
	})
}

// Registry maps kind names to handlers.
type Registry struct {
	handlers map[string]func(context.Context, []byte) error
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]func(context.Context, []byte) error)}
}

// Handle registers the handler for kind k. Registering a kind twice is a
// programming error and panics at startup.
func Handle[T any](r *Registry, k Kind[T], handler func(ctx context.Context, payload T) error) {
	if _, dup := r.handlers[k.Name]; dup {
		panic(fmt.Sprintf("jobs: kind %q registered twice", k.Name))
	}
	r.handlers[k.Name] = func(ctx context.Context, raw []byte) error {
		var payload T
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", k.Name, err)
		}
		return handler(ctx, payload)
	}
}

// Backoff returns the wait before retrying after the given (1-based) failed
// attempt. random must return a value in [0, 1).
func Backoff(attempt int, random func() float64) time.Duration {
	d := float64(baseDelay) * math.Pow(2, float64(attempt-1))
	d = min(d, float64(maxDelay))
	return time.Duration(d * (0.75 + 0.5*random()))
}

// Permanent marks a handler error as one that retrying cannot fix, such as a
// mail server rejecting an address that does not exist. The job fails at once
// instead of using its remaining attempts.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

type permanentError struct{ err error }

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// IsPermanent reports whether err was marked with Permanent.
func IsPermanent(err error) bool {
	var p *permanentError
	return errors.As(err, &p)
}
