// Package password hashes and verifies account passwords and enforces the
// password policy.
//
// Hashes use argon2id, a memory-hard function: each guess costs an attacker
// the configured memory as well as time, which blunts GPU cracking. Hashes are
// stored in the PHC string format, which records the parameters alongside the
// salt and hash:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
//
// Recording the parameters lets old hashes keep verifying after the defaults
// change, and lets callers detect and upgrade them at the next login.
package password

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Current parameters: the OWASP Password Storage Cheat Sheet's argon2id
// recommendation (19 MiB memory, 2 iterations, 1 degree of parallelism).
const (
	memoryKiB   = 19 * 1024
	iterations  = 2
	parallelism = 1
	saltLength  = 16
	keyLength   = 32
)

// Policy limits, counted in characters (not bytes).
const (
	MinLength = 12
	MaxLength = 128
)

var (
	ErrTooShort   = fmt.Errorf("password must be at least %d characters", MinLength)
	ErrTooLong    = fmt.Errorf("password must be at most %d characters", MaxLength)
	ErrCommon     = errors.New("password is too commonly used; choose another")
	ErrMalformed  = errors.New("malformed password hash")
	b64           = base64.RawStdEncoding
	phcAlgorithm  = "argon2id"
	phcVersionTag = fmt.Sprintf("v=%d", argon2.Version)
)

//go:embed common-passwords.txt
var commonList []byte

var common = func() map[string]struct{} {
	set := make(map[string]struct{})
	sc := bufio.NewScanner(bytes.NewReader(commonList))
	for sc.Scan() {
		if line := sc.Text(); line != "" && !strings.HasPrefix(line, "#") {
			// Lowercase here (Unicode-aware), matching CheckPolicy, rather than
			// trusting the file's own casing.
			set[strings.ToLower(line)] = struct{}{}
		}
	}
	return set
}()

// CheckPolicy reports whether pw may be used as a password. The rules follow
// NIST SP 800-63B: a length range and a list of known-common passwords, with
// no composition rules ("must contain a digit") that push people toward
// predictable patterns.
func CheckPolicy(pw string) error {
	n := utf8.RuneCountInString(pw)
	switch {
	case n < MinLength:
		return ErrTooShort
	case n > MaxLength:
		return ErrTooLong
	}
	if _, ok := common[strings.ToLower(pw)]; ok {
		return ErrCommon
	}
	return nil
}

// Hash returns the PHC-encoded argon2id hash of pw with a fresh random salt.
func Hash(pw string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(pw), salt, iterations, memoryKiB, parallelism, keyLength)
	return fmt.Sprintf("$%s$%s$m=%d,t=%d,p=%d$%s$%s",
		phcAlgorithm, phcVersionTag, memoryKiB, iterations, parallelism, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify reports whether pw matches encoded. needsRehash is true when the hash
// was made with parameters other than the current ones; after a successful
// login the caller should store a fresh Hash(pw).
func Verify(pw, encoded string) (ok, needsRehash bool, err error) {
	p, err := parse(encoded)
	if err != nil {
		return false, false, err
	}
	key := argon2.IDKey([]byte(pw), p.salt, p.iterations, p.memoryKiB, p.parallelism, uint32(len(p.key)))
	// Constant-time comparison: the time taken must not reveal how many
	// leading bytes matched.
	if subtle.ConstantTimeCompare(key, p.key) != 1 {
		return false, false, nil
	}
	current := p.memoryKiB == memoryKiB && p.iterations == iterations && p.parallelism == parallelism &&
		len(p.salt) == saltLength && len(p.key) == keyLength
	return true, !current, nil
}

type params struct {
	memoryKiB, iterations uint32
	parallelism           uint8
	salt, key             []byte
}

func parse(encoded string) (params, error) {
	// "$argon2id$v=19$m=..,t=..,p=..$salt$key" splits into 6 parts, the first empty.
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != phcAlgorithm || parts[2] != phcVersionTag {
		return params{}, ErrMalformed
	}
	var p params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memoryKiB, &p.iterations, &p.parallelism); err != nil {
		return params{}, ErrMalformed
	}
	var err error
	if p.salt, err = b64.DecodeString(parts[4]); err != nil {
		return params{}, ErrMalformed
	}
	if p.key, err = b64.DecodeString(parts[5]); err != nil {
		return params{}, ErrMalformed
	}
	// Reject values that are nonsensical or would let a stored hash force
	// excessive work (1 GiB memory, 10 iterations).
	if p.memoryKiB == 0 || p.memoryKiB > 1<<20 || p.iterations == 0 || p.iterations > 10 ||
		p.parallelism == 0 || len(p.salt) < 8 || len(p.key) < 16 {
		return params{}, ErrMalformed
	}
	return p, nil
}
