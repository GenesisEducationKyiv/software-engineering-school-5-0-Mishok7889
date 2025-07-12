package ports

import "context"

// EmailParams represents parameters for sending emails
type EmailParams struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// EmailProvider defines the contract for email sending
type EmailProvider interface {
	SendEmail(ctx context.Context, params EmailParams) error
}
