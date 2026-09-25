// Package config loads server configuration from environment variables.
package config

import "errors"

// Config holds settings needed to start the server.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string
	// WebDir is the directory containing the built frontend (Vite's dist output).
	WebDir string
	// DatabaseURL is the PostgreSQL connection string. It has no default:
	// connecting to an unintended database is worse than failing to start.
	DatabaseURL string
}

// Load reads configuration using getenv (normally os.Getenv), applying defaults
// for unset optional values. Taking getenv as a parameter keeps tests free of
// global state.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:        valueOr(getenv("AMETHYST_ADDR"), ":8080"),
		WebDir:      valueOr(getenv("AMETHYST_WEB_DIR"), "web/dist"),
		DatabaseURL: getenv("AMETHYST_DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("AMETHYST_DATABASE_URL is required")
	}
	return cfg, nil
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
