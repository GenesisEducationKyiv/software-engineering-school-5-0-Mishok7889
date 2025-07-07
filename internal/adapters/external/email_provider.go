package external

import (
	"context"
	"fmt"
	"net/smtp"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// SMTPEmailProviderAdapter implements EmailProvider port using SMTP
type SMTPEmailProviderAdapter struct {
	host     string
	port     int
	username string
	password string
	fromName string
	fromAddr string
}

// EmailProviderConfig represents SMTP configuration
type EmailProviderConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	FromName string
	FromAddr string
}

// NewSMTPEmailProviderAdapter creates a new SMTP email provider adapter
func NewSMTPEmailProviderAdapter(config EmailProviderConfig) ports.EmailProvider {
	return &SMTPEmailProviderAdapter{
		host:     config.Host,
		port:     config.Port,
		username: config.Username,
		password: config.Password,
		fromName: config.FromName,
		fromAddr: config.FromAddr,
	}
}

// SendEmail sends an email using SMTP
func (p *SMTPEmailProviderAdapter) SendEmail(ctx context.Context, params ports.EmailParams) error {
	if params.To == "" {
		return errors.NewValidationError("recipient email cannot be empty")
	}
	if params.Subject == "" {
		return errors.NewValidationError("email subject cannot be empty")
	}
	if params.Body == "" {
		return errors.NewValidationError("email body cannot be empty")
	}

	auth := smtp.PlainAuth("", p.username, p.password, p.host)

	from := fmt.Sprintf("%s <%s>", p.fromName, p.fromAddr)
	msg := p.buildMessage(from, params.To, params.Subject, params.Body, params.IsHTML)

	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	if err := smtp.SendMail(addr, auth, p.fromAddr, []string{params.To}, []byte(msg)); err != nil {
		return errors.NewEmailError("failed to send email", err)
	}

	return nil
}

// ValidateConfiguration validates the email provider configuration
func (p *SMTPEmailProviderAdapter) ValidateConfiguration() error {
	if p.host == "" {
		return errors.NewConfigurationError("SMTP host cannot be empty", nil)
	}
	if p.port < 1 || p.port > 65535 {
		return errors.NewConfigurationError("SMTP port must be between 1 and 65535", nil)
	}
	if p.fromAddr == "" {
		return errors.NewConfigurationError("from address cannot be empty", nil)
	}
	if p.fromName == "" {
		return errors.NewConfigurationError("from name cannot be empty", nil)
	}
	return nil
}

// buildMessage constructs the email message
func (p *SMTPEmailProviderAdapter) buildMessage(from, to, subject, body string, isHTML bool) string {
	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	msg := fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType)
	msg += "\r\n"
	msg += body

	return msg
}
