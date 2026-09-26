package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
)

// withServer returns a pool and the id of a server row to hang accounts on.
func withServer(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	pool := dbtest.New(t)
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO servers (canonical_origin, is_local) VALUES ('https://a.example', true) RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return pool, id
}

func insertAccount(pool *pgxpool.Pool, serverID, username string) error {
	_, err := pool.Exec(context.Background(),
		`INSERT INTO accounts (id, home_server_id, canonical_url, username)
		 VALUES (uuidv7(), $1, 'https://a.example/accounts/' || gen_random_uuid(), $2)`, serverID, username)
	return err
}

func TestAccountsRejectsMalformedUsernames(t *testing.T) {
	pool, server := withServer(t)
	for _, username := range []string{"Alice", "al", "1alice", "_alice", "alice-b", "alice b", "abcdefghijklmnopqrstuvwxyz12345"} {
		t.Run(username, func(t *testing.T) {
			requireSQLState(t, insertAccount(pool, server, username), checkViolation)
		})
	}
}

func TestAccountsUsernameUniquePerHomeServer(t *testing.T) {
	ctx := context.Background()
	pool, a := withServer(t)
	if err := insertAccount(pool, a, "alice"); err != nil {
		t.Fatal(err)
	}
	requireSQLState(t, insertAccount(pool, a, "alice"), uniqueViolation)

	// The same username on another server is a different account.
	var b string
	if err := pool.QueryRow(ctx, `INSERT INTO servers (canonical_origin) VALUES ('https://b.example') RETURNING id`).Scan(&b); err != nil {
		t.Fatal(err)
	}
	if err := insertAccount(pool, b, "alice"); err != nil {
		t.Errorf("alice on another server: %v", err)
	}
}

func TestLocalAccountsEmailIsCaseInsensitivelyUnique(t *testing.T) {
	ctx := context.Background()
	pool, server := withServer(t)
	for _, u := range []string{"alice", "bob"} {
		if err := insertAccount(pool, server, u); err != nil {
			t.Fatal(err)
		}
	}
	insertLocal := func(username, email string) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO local_accounts (account_id, email, password_hash)
			 SELECT id, $2, '$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA' FROM accounts WHERE username = $1`, username, email)
		return err
	}

	if err := insertLocal("alice", "Alice@Example.org"); err != nil {
		t.Fatal(err)
	}
	requireSQLState(t, insertLocal("bob", "alice@example.ORG"), uniqueViolation)
}

func TestLocalAccountsRejectsNonArgon2Hashes(t *testing.T) {
	ctx := context.Background()
	pool, server := withServer(t)
	if err := insertAccount(pool, server, "alice"); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO local_accounts (account_id, email, password_hash)
		 SELECT id, 'a@example.org', 'plaintext-password' FROM accounts WHERE username = 'alice'`)
	requireSQLState(t, err, checkViolation)
}
