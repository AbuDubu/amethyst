package migrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
)

// PostgreSQL error codes: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	uniqueViolation = "23505"
	checkViolation  = "23514"
)

func requireSQLState(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("err = %v, want PostgreSQL error %s", err, code)
	}
	if pgErr.Code != code {
		t.Fatalf("SQLSTATE = %s (%s), want %s", pgErr.Code, pgErr.Message, code)
	}
}

func TestServersAllowsAtMostOneLocalServer(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	if _, err := pool.Exec(ctx, `INSERT INTO servers (canonical_origin, is_local) VALUES ('https://a.example', true)`); err != nil {
		t.Fatalf("first local server: %v", err)
	}
	_, err := pool.Exec(ctx, `INSERT INTO servers (canonical_origin, is_local) VALUES ('https://b.example', true)`)
	requireSQLState(t, err, uniqueViolation)

	// Any number of remote servers is fine.
	if _, err := pool.Exec(ctx, `INSERT INTO servers (canonical_origin) VALUES ('https://c.example'), ('https://d.example')`); err != nil {
		t.Fatalf("remote servers: %v", err)
	}
}

func TestServersRejectsDuplicateOrigin(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	if _, err := pool.Exec(ctx, `INSERT INTO servers (canonical_origin) VALUES ('https://a.example')`); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, `INSERT INTO servers (canonical_origin) VALUES ('https://a.example')`)
	requireSQLState(t, err, uniqueViolation)
}

func TestServersRejectsNonCanonicalOrigins(t *testing.T) {
	for _, origin := range []string{
		"https://A.example",         // uppercase host
		"HTTPS://a.example",         // uppercase scheme
		"https://a.example/",        // trailing slash
		"https://a.example/path",    // path
		"a.example",                 // no scheme
		"ftp://a.example",           // unsupported scheme
		"https://a.example:443:443", // malformed port
	} {
		t.Run(origin, func(t *testing.T) {
			_, err := dbtest.New(t).Exec(context.Background(),
				`INSERT INTO servers (canonical_origin) VALUES ($1)`, origin)
			requireSQLState(t, err, checkViolation)
		})
	}
}

func TestServersGeneratesUUIDv7IDs(t *testing.T) {
	var version int
	err := dbtest.New(t).QueryRow(context.Background(),
		`INSERT INTO servers (canonical_origin) VALUES ('http://localhost:8080') RETURNING uuid_extract_version(id)`,
	).Scan(&version)
	if err != nil {
		t.Fatal(err)
	}
	if version != 7 {
		t.Errorf("UUID version = %d, want 7", version)
	}
}
