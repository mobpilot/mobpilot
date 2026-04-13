package ports

import "context"

// EmailMessage is a simple transactional email payload.
type EmailMessage struct {
	To      string
	Subject string
	Body    string // plain text
}

// EmailPort sends transactional emails.
type EmailPort interface {
	Send(ctx context.Context, msg EmailMessage) error
}
