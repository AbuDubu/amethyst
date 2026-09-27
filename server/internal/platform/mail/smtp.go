package mail

import (
	"context"
	"errors"
	"fmt"
	"time"

	gomail "github.com/wneessen/go-mail"

	"github.com/AbuDubu/amethyst/server/internal/platform/config"
	"github.com/AbuDubu/amethyst/server/internal/platform/jobs"
)

const smtpTimeout = 30 * time.Second

// SMTP delivers messages to a mail server.
type SMTP struct {
	cfg config.Mail
}

func NewSMTP(cfg config.Mail) *SMTP {
	return &SMTP{cfg: cfg}
}

// Send delivers m. A 5xx rejection from the server (such as an unknown
// mailbox) is marked permanent; connection problems and 4xx replies are left
// to the queue's retries.
func (s *SMTP) Send(ctx context.Context, m Message) error {
	msg := gomail.NewMsg()
	if err := msg.From(s.cfg.From); err != nil {
		return jobs.Permanent(fmt.Errorf("sender: %w", err))
	}
	if err := msg.To(m.To); err != nil {
		return jobs.Permanent(fmt.Errorf("recipient: %w", err))
	}
	msg.Subject(m.Subject)
	msg.SetMessageIDWithValue(m.MessageID)
	msg.SetDate()
	msg.SetBodyString(gomail.TypeTextPlain, m.Text)

	client, err := gomail.NewClient(s.cfg.Host, s.options()...)
	if err != nil {
		return jobs.Permanent(fmt.Errorf("smtp client: %w", err))
	}
	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		var sendErr *gomail.SendError
		if errors.As(err, &sendErr) && sendErr.ErrorCode() >= 500 && sendErr.ErrorCode() < 600 {
			return jobs.Permanent(fmt.Errorf("smtp rejected message: %w", err))
		}
		return fmt.Errorf("smtp: %w", err)
	}
	return nil
}

func (s *SMTP) options() []gomail.Option {
	opts := []gomail.Option{gomail.WithPort(s.cfg.Port), gomail.WithTimeout(smtpTimeout)}
	switch s.cfg.TLS {
	case config.MailTLSImplicit:
		opts = append(opts, gomail.WithSSL())
	case config.MailTLSNone:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	default:
		// Never fall back to plaintext if the server does not offer STARTTLS.
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	if s.cfg.Username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(s.cfg.Username),
			gomail.WithPassword(s.cfg.Password))
	}
	return opts
}
