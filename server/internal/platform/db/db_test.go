package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AbuDubu/amethyst/server/internal/platform/db"
	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
)

func TestCheckCurrentReportsPendingMigrations(t *testing.T) {
	ctx := context.Background()
	m, err := db.NewMigrator(dbtest.NewEmpty(t))
	if err != nil {
		t.Fatal(err)
	}

	if err := m.CheckCurrent(ctx); !errors.Is(err, db.ErrMigrationsPending) {
		t.Fatalf("CheckCurrent on empty database = %v, want ErrMigrationsPending", err)
	}

	applied, err := m.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("Up applied nothing to an empty database")
	}
	if err := m.CheckCurrent(ctx); err != nil {
		t.Fatalf("CheckCurrent after Up = %v, want nil", err)
	}
}

func TestUpIsIdempotent(t *testing.T) {
	m, err := db.NewMigrator(dbtest.New(t))
	if err != nil {
		t.Fatal(err)
	}

	applied, err := m.Up(context.Background())
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("Up on a migrated database applied %v, want nothing", applied)
	}
}

func TestReady(t *testing.T) {
	ctx := context.Background()

	t.Run("migrated database is ready", func(t *testing.T) {
		pool := dbtest.New(t)
		m, err := db.NewMigrator(pool)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Ready(pool, m)(ctx); err != nil {
			t.Errorf("Ready = %v, want nil", err)
		}
	})

	t.Run("unmigrated database is not ready", func(t *testing.T) {
		pool := dbtest.NewEmpty(t)
		m, err := db.NewMigrator(pool)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Ready(pool, m)(ctx); !errors.Is(err, db.ErrMigrationsPending) {
			t.Errorf("Ready = %v, want ErrMigrationsPending", err)
		}
	})

	t.Run("unreachable database is not ready", func(t *testing.T) {
		pool := dbtest.New(t)
		m, err := db.NewMigrator(pool)
		if err != nil {
			t.Fatal(err)
		}
		pool.Close() // Every later query fails, as during an outage.
		if err := db.Ready(pool, m)(ctx); err == nil {
			t.Error("Ready = nil after the pool closed, want error")
		}
	})
}
