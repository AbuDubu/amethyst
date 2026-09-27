// Package mail sends plain-text email reliably through the job queue.
//
// Features compose a Message and hand it to an Outbox inside their own
// transaction; a job then delivers it with retries. Because messages may carry
// secret links, the job kind is sensitive (its payload is deleted if delivery
// fails permanently), and nothing here logs a recipient address or body.
package mail

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/AbuDubu/amethyst/server/internal/platform/jobs"
)

// Message is a plain-text email ready to send.
type Message struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	// Purpose says why the message exists (such as "verification"). Logs use
	// it instead of the recipient or content.
	Purpose string `json:"purpose"`
	// MessageID is assigned at enqueue time and reused on every retry, so a
	// message delivered twice after a crash carries the same Message-ID and
	// many mail systems discard the duplicate.
	MessageID string `json:"message_id"`
}

// Mailer delivers one message.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// SendEmail is the delivery job.
var SendEmail = jobs.Kind[Message]{
	Name:    "email.send",
	Options: jobs.Options{Sensitive: true},
}

// Register installs the delivery handler for SendEmail.
func Register(r *jobs.Registry, m Mailer, logger *slog.Logger) {
	jobs.Handle(r, SendEmail, func(ctx context.Context, msg Message) error {
		if err := m.Send(ctx, msg); err != nil {
			return err
		}
		logger.Info("email sent", "purpose", msg.Purpose)
		return nil
	})
}

// Outbox queues messages for delivery.
type Outbox struct {
	domain string
}

// NewOutbox returns an Outbox whose Message-IDs use the domain of from.
func NewOutbox(from string) (*Outbox, error) {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("sender address: %w", err)
	}
	return &Outbox{domain: addr.Address[strings.LastIndex(addr.Address, "@")+1:]}, nil
}

// Enqueue validates m and queues it in db, which should be the transaction
// making the change the message is about.
func (o *Outbox) Enqueue(ctx context.Context, db jobs.DB, m Message) error {
	if err := validate(m); err != nil {
		return err
	}
	if m.MessageID == "" {
		m.MessageID = uuid.NewString() + "@" + o.domain
	}
	return SendEmail.Enqueue(ctx, db, m)
}

func validate(m Message) error {
	addr, err := mail.ParseAddress(m.To)
	switch {
	case err != nil || addr.Address != m.To:
		return errors.New("mail: recipient must be a bare email address")
	case m.Subject == "" || m.Text == "" || m.Purpose == "":
		return errors.New("mail: subject, text, and purpose are required")
	// The SMTP library encodes headers safely; rejecting line breaks here as
	// well means a bug upstream cannot become a header-injection hole.
	case strings.ContainsAny(m.Subject, "\r\n"):
		return errors.New("mail: subject must be a single line")
	}
	return nil
}
