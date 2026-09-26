package config

import "testing"

const (
	testDatabaseURL = "postgres://u:p@localhost:5432/db"
	testOrigin      = "http://localhost:8080"
)

// required returns the variables Load cannot do without.
func required(k string) string {
	switch k {
	case "AMETHYST_DATABASE_URL":
		return testDatabaseURL
	case "AMETHYST_CANONICAL_ORIGIN":
		return testOrigin
	}
	return ""
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(required)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.WebDir != "web/dist" {
		t.Errorf("WebDir = %q, want %q", cfg.WebDir, "web/dist")
	}
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"AMETHYST_ADDR":             "127.0.0.1:9000",
		"AMETHYST_WEB_DIR":          "/srv/web",
		"AMETHYST_DATABASE_URL":     testDatabaseURL,
		"AMETHYST_CANONICAL_ORIGIN": "https://example.org",
	}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Addr != "127.0.0.1:9000" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, "127.0.0.1:9000")
	}
	if cfg.WebDir != "/srv/web" {
		t.Errorf("WebDir = %q, want %q", cfg.WebDir, "/srv/web")
	}
	if cfg.CanonicalOrigin != "https://example.org" {
		t.Errorf("CanonicalOrigin = %q, want %q", cfg.CanonicalOrigin, "https://example.org")
	}
	if cfg.DatabaseURL != testDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, testDatabaseURL)
	}
}

func TestLoadRequiresEachRequiredVariable(t *testing.T) {
	for _, missing := range []string{"AMETHYST_DATABASE_URL", "AMETHYST_CANONICAL_ORIGIN"} {
		t.Run(missing, func(t *testing.T) {
			_, err := Load(func(k string) string {
				if k == missing {
					return ""
				}
				return required(k)
			})
			if err == nil {
				t.Fatalf("Load succeeded without %s, want error", missing)
			}
		})
	}
}

func TestLoadRejectsInvalidOrigin(t *testing.T) {
	_, err := Load(func(k string) string {
		if k == "AMETHYST_CANONICAL_ORIGIN" {
			return "http://example.org"
		}
		return required(k)
	})
	if err == nil {
		t.Fatal("Load accepted a plain-http public origin, want error")
	}
}
