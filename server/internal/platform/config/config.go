// Package config loads server configuration from environment variables.
package config

// Config holds settings needed to start the server.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string
	// WebDir is the directory containing the built frontend (Vite's dist output).
	WebDir string
}

// Load reads configuration using getenv (normally os.Getenv), applying defaults
// for unset values. Taking getenv as a parameter keeps tests free of global state.
func Load(getenv func(string) string) Config {
	return Config{
		Addr:   valueOr(getenv("AMETHYST_ADDR"), ":8080"),
		WebDir: valueOr(getenv("AMETHYST_WEB_DIR"), "web/dist"),
	}
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
