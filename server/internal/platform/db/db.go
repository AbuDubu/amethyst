// Package db opens the PostgreSQL connection pool and manages schema migrations.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/AbuDubu/amethyst/server/migrations"
)

// ErrMigrationsPending means the database schema is older than this binary expects.
var ErrMigrationsPending = errors.New("database has pending migrations; run `amethyst migrate`")

// Open connects a pool to url and verifies the connection with a ping.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}

// Migrator applies and inspects the embedded migrations for one pool.
type Migrator struct {
	provider *goose.Provider
}

// NewMigrator prepares migrations for pool. Create one per pool: goose needs a
// database/sql handle, and pgx's adapter creates a new one on every call.
func NewMigrator(pool *pgxpool.Pool) (*Migrator, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(pool), migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return &Migrator{provider: provider}, nil
}

// Up applies every pending migration and returns the versions it applied.
func (m *Migrator) Up(ctx context.Context) ([]int64, error) {
	results, err := m.provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	applied := make([]int64, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Version)
	}
	return applied, nil
}

// CheckCurrent returns ErrMigrationsPending if the database is behind the
// embedded migrations, or another error if the check itself fails.
func (m *Migrator) CheckCurrent(ctx context.Context) error {
	pending, err := m.provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("check migration status: %w", err)
	}
	if pending {
		return ErrMigrationsPending
	}
	return nil
}
