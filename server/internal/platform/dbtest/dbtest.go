// Package dbtest gives each test its own real PostgreSQL database.
//
// The embedded migrations are applied once into a template database whose name
// is derived from the migration contents. Each test then receives a cheap copy
// (CREATE DATABASE ... TEMPLATE), so tests are fully isolated, may commit real
// transactions, and never see one another's rows. The copy is dropped when the
// test ends.
//
// Tests fail, rather than skip, when no database is configured: a skipped
// integration test looks exactly like a passing one.
package dbtest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sort"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AbuDubu/amethyst/server/internal/platform/db"
	"github.com/AbuDubu/amethyst/server/migrations"
)

// EnvAdminURL names the variable holding a connection string for a role that
// may create databases. `make test` sets it to the Compose PostgreSQL.
const EnvAdminURL = "AMETHYST_TEST_DATABASE_URL"

// templateLockKey serializes template creation across concurrently running
// test binaries (go test runs packages in parallel processes).
const templateLockKey = 7_420_001

var (
	templateOnce sync.Once
	templateName string
	templateErr  error
)

// New returns a pool connected to a fresh, fully migrated database.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	templateOnce.Do(func() { templateName, templateErr = ensureTemplate(adminURL(t)) })
	if templateErr != nil {
		t.Fatalf("dbtest: prepare migrated template: %v", templateErr)
	}
	return create(t, templateName)
}

// NewEmpty returns a pool connected to a fresh database with no migrations
// applied, for testing migration behavior itself.
func NewEmpty(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return create(t, "template0")
}

func adminURL(t testing.TB) string {
	t.Helper()
	u := os.Getenv(EnvAdminURL)
	if u == "" {
		t.Fatalf("dbtest: %s is not set; run tests with `make test` (which starts PostgreSQL) or set it yourself", EnvAdminURL)
	}
	return u
}

func create(t testing.TB, template string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	admin := adminURL(t)
	name := "amethyst_test_" + randomHex(t)

	if err := exec(ctx, admin, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{name}.Sanitize(), pgx.Identifier{template}.Sanitize())); err != nil {
		t.Fatalf("dbtest: create database: %v", err)
	}
	t.Cleanup(func() {
		// FORCE disconnects anything the test leaked.
		if err := exec(ctx, admin, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", pgx.Identifier{name}.Sanitize())); err != nil {
			t.Errorf("dbtest: drop database %s: %v", name, err)
		}
	})

	pool, err := db.Open(ctx, withDatabase(t, admin, name))
	if err != nil {
		t.Fatalf("dbtest: %v", err)
	}
	t.Cleanup(pool.Close) // Runs before the drop above (cleanups are LIFO).
	return pool
}

// ensureTemplate returns the name of a database containing exactly the current
// migrations, building it if no test binary has done so yet.
func ensureTemplate(admin string) (string, error) {
	ctx := context.Background()
	sum, err := migrationsHash()
	if err != nil {
		return "", err
	}
	name := "amethyst_tmpl_" + sum

	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return "", fmt.Errorf("connect as admin: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", templateLockKey); err != nil {
		return "", err
	}
	// The lock is also released when the connection closes, so an error here is harmless.
	defer func() { _, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", templateLockKey) }()

	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return name, nil
	}

	// Build under a temporary name and rename when complete, so a crash midway
	// can never leave a half-migrated database under the real template name.
	building := name + "_building"
	if _, err := conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgx.Identifier{building}.Sanitize())); err != nil {
		return "", err
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE template0", pgx.Identifier{building}.Sanitize())); err != nil {
		return "", err
	}
	buildingURL, err := replaceDatabase(admin, building)
	if err != nil {
		return "", err
	}
	if err := migrate(ctx, buildingURL); err != nil {
		return "", err
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s",
		pgx.Identifier{building}.Sanitize(), pgx.Identifier{name}.Sanitize())); err != nil {
		return "", err
	}
	return name, nil
}

func migrate(ctx context.Context, url string) error {
	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	// Every connection must close before the database can be used as a template.
	defer pool.Close()
	m, err := db.NewMigrator(pool)
	if err != nil {
		return err
	}
	_, err = m.Up(ctx)
	return err
}

// migrationsHash fingerprints the migration files so a schema change
// automatically produces a new template instead of reusing a stale one.
func migrationsHash() (string, error) {
	var names []string
	if err := fs.WalkDir(migrations.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, path)
		}
		return err
	}); err != nil {
		return "", err
	}
	sort.Strings(names)

	h := sha256.New()
	for _, n := range names {
		b, err := fs.ReadFile(migrations.FS, n)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", n, len(b))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func exec(ctx context.Context, url, sql string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, sql)
	return err
}

func withDatabase(t testing.TB, rawURL, name string) string {
	t.Helper()
	u, err := replaceDatabase(rawURL, name)
	if err != nil {
		t.Fatalf("dbtest: %v", err)
	}
	return u
}

func replaceDatabase(rawURL, name string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", EnvAdminURL, err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

func randomHex(t testing.TB) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}
