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
