package identity_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AbuDubu/amethyst/server/internal/federation"
	fedstore "github.com/AbuDubu/amethyst/server/internal/federation/store"
	"github.com/AbuDubu/amethyst/server/internal/identity"
	"github.com/AbuDubu/amethyst/server/internal/identity/password"
	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
)

const origin = "https://a.example"

func setup(t *testing.T) (*pgxpool.Pool, *identity.Service) {
	t.Helper()
	pool := dbtest.New(t)
	local, err := federation.EnsureLocalServer(context.Background(), fedstore.New(pool), origin)
	if err != nil {
		t.Fatal(err)
	}
	return pool, identity.NewService(identity.LocalServer{ID: local.ID, Origin: local.CanonicalOrigin})
}

func alice() identity.NewLocalAccount {
	return identity.NewLocalAccount{
		Username: "Alice",
		Email:    "Alice@Example.org",
		Password: "correct horse battery staple",
		Role:     identity.RoleMember,
	}
}

func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateLocalAccount(t *testing.T) {
	ctx := context.Background()
	pool, svc := setup(t)

	acct, err := svc.CreateLocalAccount(ctx, pool, alice())
	if err != nil {
		t.Fatalf("CreateLocalAccount: %v", err)
	}

	if acct.Username != "alice" {
		t.Errorf("Username = %q, want normalized alice", acct.Username)
	}
	if want := origin + "/accounts/" + acct.ID.String(); acct.CanonicalURL != want {
		t.Errorf("CanonicalURL = %q, want ID-based %q", acct.CanonicalURL, want)
	}
	if acct.ID.Version() != 7 {
		t.Errorf("ID version = %d, want 7", acct.ID.Version())
	}
	if acct.Email != "Alice@Example.org" {
		t.Errorf("Email = %q, want as entered", acct.Email)
	}
	if acct.EmailVerifiedAt != nil || acct.Role != identity.RoleMember {
		t.Errorf("new account = %+v, want unverified member", acct)
	}

	// The stored hash verifies the password and is not the password itself.
	var hash string
	if err := pool.QueryRow(ctx, "SELECT password_hash FROM local_accounts WHERE account_id = $1", acct.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if ok, _, err := password.Verify("correct horse battery staple", hash); err != nil || !ok {
		t.Errorf("stored hash does not verify the password (ok %v, err %v)", ok, err)
	}
	if strings.Contains(hash, "correct horse") {
		t.Error("password stored in plain text")
	}
}

func TestCreateLocalAccountRejectsInvalidInputWithoutWriting(t *testing.T) {
	ctx := context.Background()
	pool, svc := setup(t)

	for field, mutate := range map[string]func(*identity.NewLocalAccount){
		"username": func(a *identity.NewLocalAccount) { a.Username = "admin" },
		"email":    func(a *identity.NewLocalAccount) { a.Email = "Alice <alice@example.org>" },
		"password": func(a *identity.NewLocalAccount) { a.Password = "qwerty123456" },
	} {
		t.Run(field, func(t *testing.T) {
			in := alice()
			mutate(&in)
			_, err := svc.CreateLocalAccount(ctx, pool, in)

			var ve *identity.ValidationError
			if !errors.As(err, &ve) || ve.Field != field {
				t.Fatalf("err = %v, want ValidationError on %s", err, field)
			}
		})
	}
	if n := count(t, pool, "accounts"); n != 0 {
		t.Errorf("accounts = %d after rejected input, want 0", n)
	}
}

func TestCreateLocalAccountConflicts(t *testing.T) {
	ctx := context.Background()
	pool, svc := setup(t)
	if _, err := svc.CreateLocalAccount(ctx, pool, alice()); err != nil {
		t.Fatal(err)
	}

	sameName := alice()
	sameName.Username = "ALICE"
	sameName.Email = "other@example.org"
	if _, err := svc.CreateLocalAccount(ctx, pool, sameName); !errors.Is(err, identity.ErrUsernameTaken) {
		t.Errorf("same username, different case: err = %v, want ErrUsernameTaken", err)
	}

	sameEmail := alice()
	sameEmail.Username = "alice2"
	sameEmail.Email = "alice@EXAMPLE.org"
	if _, err := svc.CreateLocalAccount(ctx, pool, sameEmail); !errors.Is(err, identity.ErrEmailTaken) {
		t.Errorf("same email, different case: err = %v, want ErrEmailTaken", err)
	}

	// The email conflict happened after the account row was inserted; the
	// transaction must have undone it.
	if n := count(t, pool, "accounts"); n != 1 {
		t.Errorf("accounts = %d, want 1 (failed registrations must leave nothing behind)", n)
	}
}

func TestCreateLocalAccountJoinsCallersTransaction(t *testing.T) {
	ctx := context.Background()
	pool, svc := setup(t)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateLocalAccount(ctx, tx, alice()); err != nil {
		t.Fatal(err)
	}
	// The caller decides: rolling back its transaction removes the account.
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, pool, "accounts"); n != 0 {
		t.Errorf("accounts = %d after caller rolled back, want 0", n)
	}
}
