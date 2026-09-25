package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load(func(string) string { return "" })

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.WebDir != "web/dist" {
		t.Errorf("WebDir = %q, want %q", cfg.WebDir, "web/dist")
	}
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"AMETHYST_ADDR":    "127.0.0.1:9000",
		"AMETHYST_WEB_DIR": "/srv/web",
	}
	cfg := Load(func(k string) string { return env[k] })

	if cfg.Addr != "127.0.0.1:9000" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, "127.0.0.1:9000")
	}
	if cfg.WebDir != "/srv/web" {
		t.Errorf("WebDir = %q, want %q", cfg.WebDir, "/srv/web")
	}
}
