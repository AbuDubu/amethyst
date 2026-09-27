package mail_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/AbuDubu/amethyst/server/internal/platform/config"
	"github.com/AbuDubu/amethyst/server/internal/platform/jobs"
	"github.com/AbuDubu/amethyst/server/internal/platform/mail"
)

// These tests send through real SMTP to Mailpit and read the result back
// through Mailpit's API. Like the database tests, they fail rather than skip
// when Mailpit is not configured.
const envMailpit = "AMETHYST_TEST_MAILPIT_HOST"

func mailpitHost(t *testing.T) string {
	t.Helper()
	h := os.Getenv(envMailpit)
	if h == "" {
		t.Fatalf("%s is not set; run tests with `make test` (which starts Mailpit)", envMailpit)
	}
	return h
}

func smtpTo(host string, port int) *mail.SMTP {
	return mail.NewSMTP(config.Mail{Host: host, Port: port, TLS: config.MailTLSNone, From: "Amethyst <noreply@a.example>"})
}

type mailpitMessage struct {
	ID        string
	MessageID string
	Subject   string
	From      struct{ Address, Name string }
	To, Bcc   []struct{ Address string }
	Text      string
}

// fetch waits for the message with the given subject to arrive in Mailpit.
func fetch(t *testing.T, host, subject string) mailpitMessage {
	t.Helper()
	api := "http://" + host + ":8025/api/v1"
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var found struct{ Messages []mailpitMessage }
		getJSON(t, api+"/search?query="+url.QueryEscape(fmt.Sprintf("subject:%q", subject)), &found)
		if len(found.Messages) == 1 {
			var full mailpitMessage
			getJSON(t, api+"/message/"+found.Messages[0].ID, &full)
			return full
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("message %q did not arrive in Mailpit", subject)
	return mailpitMessage{}
}

func getJSON(t *testing.T, u string, v any) {
	t.Helper()
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func TestSMTPDeliversPlainText(t *testing.T) {
	host := mailpitHost(t)
	subject := "Verify " + uuid.NewString()
	id := uuid.NewString() + "@a.example"

	err := smtpTo(host, 1025).Send(context.Background(), mail.Message{
		To: "alice@example.org", Subject: subject, Text: "Open https://a.example/verify?t=abc\n", Purpose: "test", MessageID: id,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := fetch(t, host, subject)
	if got.From.Address != "noreply@a.example" || got.From.Name != "Amethyst" {
		t.Errorf("From = %+v, want Amethyst <noreply@a.example>", got.From)
	}
	if len(got.To) != 1 || got.To[0].Address != "alice@example.org" {
		t.Errorf("To = %+v, want alice@example.org", got.To)
	}
	if got.MessageID != id {
		t.Errorf("Message-ID = %q, want %q", got.MessageID, id)
	}
	// Email lines end in CRLF on the wire (RFC 5322), so compare without them.
	if strings.ReplaceAll(got.Text, "\r\n", "\n") != "Open https://a.example/verify?t=abc\n" {
		t.Errorf("Text = %q", got.Text)
	}
}

func TestSMTPEncodesHeadersSoLineBreaksCannotAddRecipients(t *testing.T) {
	// Bypasses Outbox validation to test the library's own defense.
	host := mailpitHost(t)
	token := uuid.NewString()
	err := smtpTo(host, 1025).Send(context.Background(), mail.Message{
		To: "alice@example.org", Subject: token + "\r\nBcc: victim@example.org", Text: "hi", Purpose: "test", MessageID: token + "@a.example",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := fetch(t, host, token)
	if len(got.Bcc) != 0 || len(got.To) != 1 {
		t.Errorf("recipients To=%+v Bcc=%+v; the subject's line break added a header", got.To, got.Bcc)
	}
}

func TestSMTPConnectionFailureIsRetryable(t *testing.T) {
	// Port 1 on loopback: nothing listens, so the connection is refused.
	err := smtpTo("127.0.0.1", 1).Send(context.Background(), mail.Message{
		To: "alice@example.org", Subject: "x", Text: "x", Purpose: "test", MessageID: "x@a.example",
	})
	if err == nil {
		t.Fatal("Send to a closed port succeeded")
	}
	if jobs.IsPermanent(err) {
		t.Errorf("connection failure marked permanent (%v); it should be retried", err)
	}
}
