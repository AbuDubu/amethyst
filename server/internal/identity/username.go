package identity

import (
	"regexp"
	"strings"
)

// usernamePattern matches the accounts.username CHECK constraint; keep the two
// in step. It is checked here first so people get a clear message instead of a
// constraint violation.
var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,29}$`)

// reservedUsernames could be mistaken for the server or its staff, or collide
// with paths and roles. Only local registration checks them.
var reservedUsernames = map[string]bool{
	"abuse": true, "account": true, "accounts": true, "admin": true, "administrator": true,
	"amethyst": true, "api": true, "federation": true, "help": true, "hostmaster": true,
	"invite": true, "login": true, "logout": true, "mod": true, "moderator": true,
	"no_reply": true, "noreply": true, "null": true, "operator": true, "postmaster": true,
	"register": true, "root": true, "security": true, "settings": true, "staff": true,
	"support": true, "system": true, "undefined": true, "webmaster": true,
}

// NormalizeUsername returns the stored form of a username as typed:
// surrounding spaces removed and lowercased, so "Alice" and "alice" are the
// same account.
func NormalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// validateUsername checks a normalized username for local registration.
func validateUsername(u string) error {
	switch {
	case len(u) < 3 || len(u) > 30:
		return &ValidationError{Field: "username", Message: "must be 3 to 30 characters"}
	case !usernamePattern.MatchString(u):
		return &ValidationError{Field: "username", Message: "must start with a letter and contain only letters, digits, and underscores"}
	case reservedUsernames[u]:
		return &ValidationError{Field: "username", Message: "is reserved"}
	}
	return nil
}
