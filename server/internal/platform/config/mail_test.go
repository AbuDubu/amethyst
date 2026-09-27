package config

import "testing"

func mailEnv(overrides map[string]string) func(string) string {
	env := map[string]string{
		"AMETHYST_SMTP_HOST": "smtp.example.org",
		"AMETHYST_MAIL_FROM": "Amethyst <noreply@example.org>",
	}
	for k, v := range overrides {
		env[k] = v
	}
	return func(k string) string { return env[k] }
}

func TestLoadMailDefaults(t *testing.T) {
	m, err := LoadMail(mailEnv(nil))
	if err != nil {
		t.Fatal(err)
	}
	if m.TLS != MailTLSStartTLS || m.Port != 587 {
		t.Errorf("defaults = %s on %d, want starttls on 587", m.TLS, m.Port)
	}
}

func TestLoadMailPortFollowsTLSMode(t *testing.T) {
	m, err := LoadMail(mailEnv(map[string]string{"AMETHYST_SMTP_TLS": "tls"}))
	if err != nil || m.Port != 465 {
		t.Errorf("implicit TLS port = %d (err %v), want 465", m.Port, err)
	}
	m, err = LoadMail(mailEnv(map[string]string{"AMETHYST_SMTP_TLS": "none", "AMETHYST_SMTP_HOST": "mailpit", "AMETHYST_SMTP_PORT": "1025"}))
	if err != nil || m.Port != 1025 {
		t.Errorf("explicit port = %d (err %v), want 1025", m.Port, err)
	}
}

func TestLoadMailAllowsNoTLSOnlyLocally(t *testing.T) {
	for host, ok := range map[string]bool{
		"localhost":        true,
		"mail.localhost":   true,
		"127.0.0.1":        true,
		"::1":              true,
		"mailpit":          true, // Compose service name
		"smtp.example.org": false,
		"10.0.0.5":         false,
		"192.168.1.20":     false,
	} {
		_, err := LoadMail(mailEnv(map[string]string{"AMETHYST_SMTP_TLS": "none", "AMETHYST_SMTP_HOST": host}))
		if (err == nil) != ok {
			t.Errorf("TLS none to %s: err = %v, want allowed=%v", host, err, ok)
		}
	}
}

func TestLoadMailRejects(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"no host":       {"AMETHYST_SMTP_HOST": ""},
		"no from":       {"AMETHYST_MAIL_FROM": ""},
		"bad from":      {"AMETHYST_MAIL_FROM": "not an address"},
		"unknown tls":   {"AMETHYST_SMTP_TLS": "ssl"},
		"bad port":      {"AMETHYST_SMTP_PORT": "70000"},
		"username only": {"AMETHYST_SMTP_USERNAME": "amethyst"},
		"password only": {"AMETHYST_SMTP_PASSWORD": "secret"},
	} {
		if _, err := LoadMail(mailEnv(env)); err == nil {
			t.Errorf("%s: accepted, want error", name)
		}
	}
}
