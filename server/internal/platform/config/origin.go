package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// validateOrigin checks that origin is a canonical server origin: scheme and
// host (optionally a port), all lowercase, with no path, query, fragment, or
// credentials. The origin is this server's permanent identity, so it is
// rejected rather than silently normalized: the operator should see exactly
// what will be stored.
//
// Plain http is accepted only for localhost and *.localhost, which never leave
// the machine; any other server must be reached over https.
func validateOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("%q is not a URL: %w", origin, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%q must start with https:// (or http:// for localhost)", origin)
	}
	if u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Opaque != "" {
		return fmt.Errorf("%q must be only a scheme and host, like https://example.org", origin)
	}
	if u.Path == "/" {
		return fmt.Errorf("%q must not end with a slash", origin)
	}
	if origin != strings.ToLower(origin) {
		return fmt.Errorf("%q must be lowercase", origin)
	}
	if u.Scheme == "http" && !isLocalhost(u.Hostname()) {
		return fmt.Errorf("%q must use https; plain http is only allowed for localhost", origin)
	}
	if port := u.Port(); port != "" {
		if _, err := net.LookupPort("tcp", port); err != nil || strings.HasPrefix(port, "0") {
			return fmt.Errorf("%q has an invalid port", origin)
		}
	}
	return nil
}

func isLocalhost(host string) bool {
	return host == "localhost" || strings.HasSuffix(host, ".localhost")
}
