package ports

import (
	"context"

	"weatherapi.app/internal/core/shared"
)

// EmailFormat represents the format of email content
type EmailFormat string

const (
	FormatText     EmailFormat = "text"
	FormatHTML     EmailFormat = "html"
	FormatMarkdown EmailFormat = "markdown"
)

// EmailParams represents parameters for sending emails
type EmailParams struct {
	To      string
	Subject string
	Body    string
	Format  EmailFormat
}

// Validate validates the email parameters
func (p EmailParams) Validate() error {
	if p.To == "" {
		return NewValidationError("recipient email cannot be empty")
	}
	if p.Subject == "" {
		return NewValidationError("email subject cannot be empty")
	}
	if p.Body == "" {
		return NewValidationError("email body cannot be empty")
	}
	if p.Format != FormatText && p.Format != FormatHTML && p.Format != FormatMarkdown {
		return NewValidationError("invalid email format")
	}
	return nil
}

// IsHTML returns true if the email format is HTML
func (p EmailParams) IsHTML() bool {
	return p.Format == FormatHTML
}

// EmailProvider defines the contract for email sending
type EmailProvider interface {
	SendEmail(ctx context.Context, req shared.EmailRequest) error
}
