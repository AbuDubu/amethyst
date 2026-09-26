package federation_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/AbuDubu/amethyst/server/internal/federation"
	"github.com/AbuDubu/amethyst/server/internal/federation/store"
	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
)

func TestEnsureLocalServerRecordsOriginOnFirstStart(t *testing.T) {
	ctx := context.Background()
	q := store.New(dbtest.New(t))

	first, err := federation.EnsureLocalServer(ctx, q, "https://a.example")
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	if first.CanonicalOrigin != "https://a.example" || !first.IsLocal {
		t.Errorf("recorded %+v, want local https://a.example", first)
	}

	again, err := federation.EnsureLocalServer(ctx, q, "https://a.example")
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("restart produced a new identity %s, want %s", again.ID, first.ID)
	}
}

func TestEnsureLocalServerRefusesChangedOrigin(t *testing.T) {
	ctx := context.Background()
	q := store.New(dbtest.New(t))
	if _, err := federation.EnsureLocalServer(ctx, q, "https://a.example"); err != nil {
		t.Fatal(err)
	}

	_, err := federation.EnsureLocalServer(ctx, q, "https://b.example")
	if !errors.Is(err, federation.ErrOriginChanged) {
		t.Fatalf("err = %v, want ErrOriginChanged", err)
	}

	local, err := q.GetLocalServer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if local.CanonicalOrigin != "https://a.example" {
		t.Errorf("stored origin became %q; a refused change must not be applied", local.CanonicalOrigin)
	}
}

func TestEnsureLocalServerConcurrentStartsCreateOneIdentity(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	q := store.New(pool)

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for range 10 {
		wg.Go(func() {
			if _, err := federation.EnsureLocalServer(ctx, q, "https://a.example"); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent start: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM servers WHERE is_local").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("local servers = %d, want 1", n)
	}
}
