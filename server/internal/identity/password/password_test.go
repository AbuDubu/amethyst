package password

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

const good = "correct horse battery staple"

func TestHashAndVerify(t *testing.T) {
	encoded, err := Hash(good)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("encoded = %q, want PHC argon2id with current parameters", encoded)
	}

	ok, rehash, err := Verify(good, encoded)
	if err != nil || !ok || rehash {
		t.Errorf("Verify(correct) = ok %v, rehash %v, err %v; want true, false, nil", ok, rehash, err)
	}

	ok, _, err = Verify(good+"!", encoded)
	if err != nil || ok {
		t.Errorf("Verify(wrong) = ok %v, err %v; want false, nil", ok, err)
	}
}

func TestHashUsesAFreshSaltEachTime(t *testing.T) {
	a, _ := Hash(good)
	b, _ := Hash(good)
	if a == b {
		t.Error("two hashes of the same password are identical; salts are not random")
	}
}

func TestVerifyFlagsOutdatedParametersForRehash(t *testing.T) {
	// A hash made with weaker (older) parameters still verifies, but asks to be upgraded.
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(good), salt, 1, 8*1024, 1, keyLength)
	old := "$argon2id$v=19$m=8192,t=1,p=1$" + b64.EncodeToString(salt) + "$" + b64.EncodeToString(key)

	ok, rehash, err := Verify(good, old)
	if err != nil || !ok || !rehash {
		t.Errorf("Verify(old params) = ok %v, rehash %v, err %v; want true, true, nil", ok, rehash, err)
	}
}

func TestVerifyRejectsMalformedHashes(t *testing.T) {
	valid, _ := Hash(good)
	for name, encoded := range map[string]string{
		"empty":            "",
		"bcrypt":           "$2a$12$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ01234",
		"argon2i":          strings.Replace(valid, "argon2id", "argon2i", 1),
		"wrong version":    strings.Replace(valid, "v=19", "v=16", 1),
		"bad params":       strings.Replace(valid, "m=19456", "m=abc", 1),
		"huge memory":      strings.Replace(valid, "m=19456", "m=99999999", 1),
		"zero iterations":  strings.Replace(valid, "t=2", "t=0", 1),
		"truncated":        valid[:len(valid)-30],
		"bad base64":       valid[:len(valid)-2] + "!!",
		"missing sections": "$argon2id$v=19$m=19456,t=2,p=1",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Verify(good, encoded); !errors.Is(err, ErrMalformed) {
				t.Errorf("err = %v, want ErrMalformed", err)
			}
		})
	}
}

func TestCheckPolicy(t *testing.T) {
	for pw, want := range map[string]error{
		good:                     nil,
		"short":                  ErrTooShort,
		"elevenchars":            ErrTooShort,
		strings.Repeat("a", 129): ErrTooLong,
		strings.Repeat("a", 128): nil,
		"qwerty123456":           ErrCommon,
		"QWERTY123456":           ErrCommon, // case-insensitive
		"1q2w3e4r5t6y":           ErrCommon,
		"пароль-очень-длинный":   nil,         // 20 characters, 37 bytes
		"日本語のパスワード":              ErrTooShort, // 9 characters, 27 bytes: counted as characters
	} {
		if got := CheckPolicy(pw); !errors.Is(got, want) {
			t.Errorf("CheckPolicy(%q) = %v, want %v", pw, got, want)
		}
	}
}

func TestCommonListLoaded(t *testing.T) {
	if len(common) < 1000 {
		t.Fatalf("common password list has %d entries; embedded file missing or misparsed", len(common))
	}
	for pw := range common {
		if strings.HasPrefix(pw, "#") {
			t.Fatalf("comment line %q was loaded as a password", pw)
		}
	}
}
