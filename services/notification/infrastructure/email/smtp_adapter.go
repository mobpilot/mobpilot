package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// SMTPAdapter sends transactional emails via a plain SMTP server.
// In development, point this at MailHog (smtp://mailhog:1025).
type SMTPAdapter struct {
	addr string // host:port, e.g. "mailhog:1025"
	from string
	auth smtp.Auth // nil for unauthenticated (dev/MailHog)
}

var _ domainports.EmailPort = (*SMTPAdapter)(nil)

// NewSMTPAdapter creates an SMTP email adapter.
// Set user/password to empty strings for unauthenticated relay (MailHog).
func NewSMTPAdapter(host, port, from, user, password string) *SMTPAdapter {
	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, password, host)
	}
	return &SMTPAdapter{addr: addr, from: from, auth: auth}
}

func (a *SMTPAdapter) Send(_ context.Context, msg domainports.EmailMessage) error {
	body := strings.Join([]string{
		"From: " + a.from,
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"Content-Type: text/plain; charset=utf-8",
		"",
		msg.Body,
	}, "\r\n")

	if err := smtp.SendMail(a.addr, a.auth, a.from, []string{msg.To}, []byte(body)); err != nil {
		return fmt.Errorf("SMTPAdapter.Send: %w", err)
	}
	return nil
}

// NoopEmailPort discards all email sends. Used when SMTP is not configured.
type NoopEmailPort struct{}

var _ domainports.EmailPort = (*NoopEmailPort)(nil)

func (*NoopEmailPort) Send(_ context.Context, _ domainports.EmailMessage) error { return nil }
