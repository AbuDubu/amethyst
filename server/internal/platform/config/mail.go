package config

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strconv"
	"strings"
)

// TLS modes for the SMTP connection.
const (
	MailTLSStartTLS = "starttls" // plain connection upgraded with STARTTLS (usually port 587)
	MailTLSImplicit = "tls"      // TLS from the first byte (usually port 465)
	MailTLSNone     = "none"     // unencrypted; local development servers only
)

// Mail holds outgoing email settings. Only serve needs them.
type Mail struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      string
	// From is the sender, either an address or `Name <address>`.
	From string
}

// LoadMail reads AMETHYST_SMTP_* and AMETHYST_MAIL_FROM.
func LoadMail(getenv func(string) string) (Mail, error) {
	m := Mail{
		Host:     getenv("AMETHYST_SMTP_HOST"),
		Username: getenv("AMETHYST_SMTP_USERNAME"),
		Password: getenv("AMETHYST_SMTP_PASSWORD"),
		TLS:      valueOr(getenv("AMETHYST_SMTP_TLS"), MailTLSStartTLS),
		From:     getenv("AMETHYST_MAIL_FROM"),
	}
	if m.Host == "" {
		return Mail{}, errors.New("AMETHYST_SMTP_HOST is required")
	}
	if m.From == "" {
		return Mail{}, errors.New("AMETHYST_MAIL_FROM is required (for example `Amethyst <noreply@example.org>`)")
	}
	if _, err := mail.ParseAddress(m.From); err != nil {
		return Mail{}, fmt.Errorf("AMETHYST_MAIL_FROM: %w", err)
	}
	if (m.Username == "") != (m.Password == "") {
		return Mail{}, errors.New("set both AMETHYST_SMTP_USERNAME and AMETHYST_SMTP_PASSWORD, or neither")
	}

	defaultPort := map[string]int{MailTLSStartTLS: 587, MailTLSImplicit: 465, MailTLSNone: 25}[m.TLS]
	if defaultPort == 0 {
		return Mail{}, fmt.Errorf("AMETHYST_SMTP_TLS must be %s, %s, or %s", MailTLSStartTLS, MailTLSImplicit, MailTLSNone)
	}
	// An unencrypted connection would expose credentials and the secret links
	// inside messages, so it is only allowed to a server on this machine or a
	// development container.
	if m.TLS == MailTLSNone && !isLocalMailHost(m.Host) {
		return Mail{}, fmt.Errorf("AMETHYST_SMTP_TLS=none is only allowed for a local host, not %q", m.Host)
	}

	m.Port = defaultPort
	if p := getenv("AMETHYST_SMTP_PORT"); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil || port < 1 || port > 65535 {
			return Mail{}, fmt.Errorf("AMETHYST_SMTP_PORT %q is not a valid port", p)
		}
		m.Port = port
	}
	return m, nil
}

// isLocalMailHost accepts loopback addresses, localhost names, and dotless
// names such as a Compose service ("mailpit"), which resolve only inside a
// private container network.
func isLocalMailHost(host string) bool {
	if isLocalhost(host) {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return !strings.Contains(host, ".")
}
