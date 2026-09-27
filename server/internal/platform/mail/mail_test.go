package mail_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AbuDubu/amethyst/server/internal/platform/dbtest"
	"github.com/AbuDubu/amethyst/server/internal/platform/jobs"
	"github.com/AbuDubu/amethyst/server/internal/platform/mail"
)

func verification() mail.Message {
	return mail.Message{
		To:      "alice@example.org",
		Subject: "Verify your email address",
		Text:    "Open this link: https://a.example/verify?token=SECRET-TOKEN",
		Purpose: "verification",
	}
}

func newOutbox(t *testing.T) *mail.Outbox {
	t.Helper()
	o, err := mail.NewOutbox("Amethyst <noreply@a.example>")
	if err != nil {
		t.Fatal(err)
	}
	return o
}

// deliver runs one worker pass with rec as the mailer, returning the log output.
func deliver(t *testing.T, pool *pgxpool.Pool, rec *mail.Recorder) string {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	r := jobs.NewRegistry()
	mail.Register(r, rec, logger)
	if _, err := jobs.NewWorker(pool, r, logger).RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	return logs.String()
}

func TestQueuedMessageIsDeliveredWithAStableMessageID(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	tx, _ := pool.Begin(ctx)
	if err := newOutbox(t).Enqueue(ctx, tx, verification()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	rec := &mail.Recorder{}
	logs := deliver(t, pool, rec)

	sent := rec.Sent()
	if len(sent) != 1 || sent[0].Subject != "Verify your email address" {
		t.Fatalf("sent = %+v, want the verification message", sent)
	}
	if !regexp.MustCompile(`^[0-9a-f-]{36}@a\.example$`).MatchString(sent[0].MessageID) {
		t.Errorf("MessageID = %q, want <uuid>@a.example", sent[0].MessageID)
	}
	// Logs name the purpose, never the recipient or the secret link.
	if !strings.Contains(logs, "purpose=verification") {
		t.Errorf("logs do not mention the purpose: %s", logs)
	}
	for _, secret := range []string{"alice@example.org", "SECRET-TOKEN"} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs contain %q: %s", secret, logs)
		}
	}
}

func TestRolledBackMessageIsNeverSent(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	tx, _ := pool.Begin(ctx)
	_ = newOutbox(t).Enqueue(ctx, tx, verification())
	_ = tx.Rollback(ctx)

	rec := &mail.Recorder{}
	deliver(t, pool, rec)
	if n := len(rec.Sent()); n != 0 {
		t.Errorf("sent %d messages for a rolled-back change, want 0", n)
	}
}

func TestPermanentDeliveryFailureScrubsTheMessage(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	_ = newOutbox(t).Enqueue(ctx, pool, verification())

	deliver(t, pool, &mail.Recorder{Err: jobs.Permanent(errors.New("550 no such mailbox"))})

	var status string
	var payload []byte
	if err := pool.QueryRow(ctx, `SELECT status, payload FROM jobs`).Scan(&status, &payload); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || payload != nil {
		t.Errorf("job = %s with payload %q, want failed with the secret link deleted", status, payload)
	}
}

func TestOutboxRejectsInvalidMessages(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	for name, mutate := range map[string]func(*mail.Message){
		"named recipient":  func(m *mail.Message) { m.To = "Alice <alice@example.org>" },
		"not an address":   func(m *mail.Message) { m.To = "alice" },
		"empty subject":    func(m *mail.Message) { m.Subject = "" },
		"empty text":       func(m *mail.Message) { m.Text = "" },
		"no purpose":       func(m *mail.Message) { m.Purpose = "" },
		"header injection": func(m *mail.Message) { m.Subject = "Hi\r\nBcc: victim@example.org" },
	} {
		m := verification()
		mutate(&m)
		if err := newOutbox(t).Enqueue(ctx, pool, m); err == nil {
			t.Errorf("%s: accepted, want error", name)
		}
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM jobs`).Scan(&n)
	if n != 0 {
		t.Errorf("%d jobs queued for invalid messages, want 0", n)
	}
}
