package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/AbuDubu/amethyst/server/internal/federation/store"
	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
)

func TestLocalServerRoundTrip(t *testing.T) {
	ctx := context.Background()
	q := store.New(dbtest.New(t))

	if _, err := q.GetLocalServer(ctx); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetLocalServer on a fresh database = %v, want pgx.ErrNoRows", err)
	}

	if _, err := q.CreateServer(ctx, store.CreateServerParams{CanonicalOrigin: "https://b.example"}); err != nil {
		t.Fatalf("create remote server: %v", err)
	}
	created, err := q.CreateServer(ctx, store.CreateServerParams{CanonicalOrigin: "https://a.example", IsLocal: true})
	if err != nil {
		t.Fatalf("create local server: %v", err)
	}

	got, err := q.GetLocalServer(ctx)
	if err != nil {
		t.Fatalf("GetLocalServer: %v", err)
	}
	if got.ID != created.ID || got.CanonicalOrigin != "https://a.example" {
		t.Errorf("GetLocalServer = %+v, want the local server %+v", got, created)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set by the database")
	}
}
