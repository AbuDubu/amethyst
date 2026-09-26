// Package identity owns accounts: local accounts registered here, their
// private credentials, and (from Milestone 4) remote accounts known from peers.
package identity

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbuDubu/amethyst/server/internal/identity/password"
	"github.com/AbuDubu/amethyst/server/internal/identity/store"
)

// Role is a local account's role on this server (not in any community).
type Role string

const (
	RoleMember   Role = "member"
	RoleOperator Role = "operator"
)

var (
	ErrUsernameTaken = errors.New("username is already taken")
	ErrEmailTaken    = errors.New("email address is already registered")
)

// ValidationError describes invalid input for one field, in terms safe to
// show the person who entered it.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + " " + e.Message }

// LocalServer identifies this server, the home of every local account.
type LocalServer struct {
	ID     uuid.UUID
	Origin string
}

// Beginner starts a transaction. Both *pgxpool.Pool and pgx.Tx satisfy it;
// on a pgx.Tx, Begin creates a savepoint, so callers can include account
// creation in their own larger transaction.
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service creates and manages local accounts.
type Service struct {
	server LocalServer
}

func NewService(server LocalServer) *Service {
	return &Service{server: server}
}

// NewLocalAccount is the input for registering a local account.
type NewLocalAccount struct {
	Username string // normalized by CreateLocalAccount
	Email    string
	Password string
	Role     Role
}

// LocalAccount is a local account as seen by its owner. It never includes the
// password hash.
type LocalAccount struct {
	ID              uuid.UUID
	CanonicalURL    string
	Username        string
	Email           string
	EmailVerifiedAt *time.Time
	Role            Role
	CreatedAt       time.Time
}

// CreateLocalAccount validates input, hashes the password, and stores the
// account and its private details atomically. Input problems are returned as
// *ValidationError; conflicts as ErrUsernameTaken or ErrEmailTaken.
func (s *Service) CreateLocalAccount(ctx context.Context, db Beginner, in NewLocalAccount) (LocalAccount, error) {
	username := NormalizeUsername(in.Username)
	email := strings.TrimSpace(in.Email)
	if err := validateUsername(username); err != nil {
		return LocalAccount{}, err
	}
	if err := validateEmail(email); err != nil {
		return LocalAccount{}, err
	}
	if err := password.CheckPolicy(in.Password); err != nil {
		return LocalAccount{}, &ValidationError{Field: "password", Message: strings.TrimPrefix(err.Error(), "password ")}
	}
	if in.Role != RoleMember && in.Role != RoleOperator {
		return LocalAccount{}, fmt.Errorf("unknown role %q", in.Role)
	}

	// Hash before opening the transaction: it is deliberately slow (~tens of
	// ms) and must not hold a database connection while it runs.
	hash, err := password.Hash(in.Password)
	if err != nil {
		return LocalAccount{}, err
	}
	// Generated here rather than by the database because the canonical URL,
	// inserted in the same statement, contains it.
	id, err := uuid.NewV7()
	if err != nil {
		return LocalAccount{}, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return LocalAccount{}, err
	}
	// Rollback after a successful Commit is a no-op, so this only undoes failures.
	defer func() { _ = tx.Rollback(ctx) }()

	q := store.New(tx)
	account, err := q.InsertAccount(ctx, store.InsertAccountParams{
		ID:           id,
		HomeServerID: s.server.ID,
		CanonicalUrl: s.server.Origin + "/accounts/" + id.String(),
		Username:     username,
	})
	if err != nil {
		return LocalAccount{}, conflictOr(err)
	}
	local, err := q.InsertLocalAccount(ctx, store.InsertLocalAccountParams{
		AccountID:    id,
		Email:        email,
		PasswordHash: hash,
		ServerRole:   string(in.Role),
	})
	if err != nil {
		return LocalAccount{}, conflictOr(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LocalAccount{}, err
	}

	return LocalAccount{
		ID:              account.ID,
		CanonicalURL:    account.CanonicalUrl,
		Username:        account.Username,
		Email:           local.Email,
		EmailVerifiedAt: local.EmailVerifiedAt,
		Role:            Role(local.ServerRole),
		CreatedAt:       account.CreatedAt,
	}, nil
}

func validateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	// Require a bare address: ParseAddress also accepts `Name <a@b>`.
	if err != nil || addr.Address != email || len(email) > 254 {
		return &ValidationError{Field: "email", Message: "must be a valid email address"}
	}
	return nil
}

// conflictOr translates unique-constraint violations into domain errors. The
// database is the final arbiter: checking for an existing username first
// would race with a concurrent registration.
func conflictOr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "accounts_home_server_id_username_key":
			return ErrUsernameTaken
		case "local_accounts_email_key":
			return ErrEmailTaken
		}
	}
	return err
}
