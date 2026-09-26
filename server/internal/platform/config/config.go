// Package config loads server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
)

// Config holds settings needed to start the server.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string
	// WebDir is the directory containing the built frontend (Vite's dist output).
	WebDir string
	// DatabaseURL is the PostgreSQL connection string. It has no default:
	// connecting to an unintended database is worse than failing to start.
	DatabaseURL string
	// CanonicalOrigin is this server's permanent public identity, such as
	// https://example.org. Other servers and accounts refer to it, so it has
	// no default and cannot change once recorded.
	CanonicalOrigin string
}

// Load reads configuration using getenv (normally os.Getenv), applying defaults
// for unset optional values. Taking getenv as a parameter keeps tests free of
// global state.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:            valueOr(getenv("AMETHYST_ADDR"), ":8080"),
		WebDir:          valueOr(getenv("AMETHYST_WEB_DIR"), "web/dist"),
		DatabaseURL:     getenv("AMETHYST_DATABASE_URL"),
		CanonicalOrigin: getenv("AMETHYST_CANONICAL_ORIGIN"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("AMETHYST_DATABASE_URL is required")
	}
	if cfg.CanonicalOrigin == "" {
		return Config{}, errors.New("AMETHYST_CANONICAL_ORIGIN is required (for example https://example.org)")
	}
	if err := validateOrigin(cfg.CanonicalOrigin); err != nil {
		return Config{}, fmt.Errorf("AMETHYST_CANONICAL_ORIGIN: %w", err)
	}
	return cfg, nil
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
