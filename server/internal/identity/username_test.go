package identity

import (
	"errors"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	if got := NormalizeUsername("  Alice_B  "); got != "alice_b" {
		t.Errorf("NormalizeUsername = %q, want alice_b", got)
	}
}

func TestValidateUsername(t *testing.T) {
	for u, ok := range map[string]bool{
		"alice":                           true,
		"bob_2":                           true,
		"abc":                             true,
		"a23456789012345678901234567890":  true, // 30 characters
		"ab":                              false,
		"a234567890123456789012345678901": false, // 31 characters
		"1alice":                          false,
		"_alice":                          false,
		"alice-b":                         false,
		"alice.b":                         false,
		"élodie":                          false,
		"admin":                           false, // reserved
		"operator":                        false, // reserved
	} {
		err := validateUsername(u)
		var ve *ValidationError
		switch {
		case ok && err != nil:
			t.Errorf("validateUsername(%q) = %v, want nil", u, err)
		case !ok && !errors.As(err, &ve):
			t.Errorf("validateUsername(%q) = %v, want *ValidationError", u, err)
		case !ok && ve.Field != "username":
			t.Errorf("validateUsername(%q) field = %q, want username", u, ve.Field)
		}
	}
}
