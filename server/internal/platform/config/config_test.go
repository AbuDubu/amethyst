package config

import "testing"

const testDatabaseURL = "postgres://u:p@localhost:5432/db"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(func(k string) string {
		if k == "AMETHYST_DATABASE_URL" {
			return testDatabaseURL
		}
		return ""
	})
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
		"AMETHYST_ADDR":         "127.0.0.1:9000",
		"AMETHYST_WEB_DIR":      "/srv/web",
		"AMETHYST_DATABASE_URL": testDatabaseURL,
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
	if cfg.DatabaseURL != testDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, testDatabaseURL)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	_, err := Load(func(string) string { return "" })

	if err == nil {
		t.Fatal("Load succeeded without AMETHYST_DATABASE_URL, want error")
	}
}
