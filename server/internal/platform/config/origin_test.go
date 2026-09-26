package config

import "testing"

func TestValidateOriginAccepts(t *testing.T) {
	for _, origin := range []string{
		"https://example.org",
		"https://social.example.org:8443",
		"http://localhost:8080",
		"http://a.localhost:8081",
		"https://a.localhost",
	} {
		if err := validateOrigin(origin); err != nil {
			t.Errorf("validateOrigin(%q) = %v, want nil", origin, err)
		}
	}
}

func TestValidateOriginRejects(t *testing.T) {
	for _, origin := range []string{
		"",
		"example.org",                  // no scheme
		"ftp://example.org",            // unsupported scheme
		"http://example.org",           // http outside localhost
		"http://localhost.example.org", // not actually localhost
		"https://Example.org",          // not lowercase
		"https://example.org/",         // trailing slash
		"https://example.org/amethyst", // path
		"https://example.org?x=1",      // query
		"https://example.org#top",      // fragment
		"https://user:pw@example.org",  // credentials
		"https://example.org:0808",     // non-canonical port
		"https://example.org:99999",    // out-of-range port
	} {
		if err := validateOrigin(origin); err == nil {
			t.Errorf("validateOrigin(%q) = nil, want error", origin)
		}
	}
}
