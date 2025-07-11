package external

import (
	"context"
	"crypto/tls"
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

// SendEmail sends an email using SMTP with flexible authentication and TLS
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

	from := fmt.Sprintf("%s <%s>", p.fromName, p.fromAddr)
	msg := p.buildMessage(from, params.To, params.Subject, params.Body, params.IsHTML)
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	// Use manual SMTP client for better control over authentication and TLS
	client, err := smtp.Dial(addr)
	if err != nil {
		return errors.NewEmailError("failed to connect to SMTP server", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			// Intentionally ignore close errors for cleanup
			_ = closeErr
		}
	}()

	// Start TLS if supported
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: p.host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return errors.NewEmailError("failed to establish secure TLS connection", err)
		}
	}

	// Authenticate only if credentials are provided
	if p.username != "" && p.password != "" {
		auth := smtp.PlainAuth("", p.username, p.password, p.host)
		if err := client.Auth(auth); err != nil {
			return errors.NewEmailError("failed to authenticate", err)
		}
	}

	// Set sender
	if err := client.Mail(p.fromAddr); err != nil {
		return errors.NewEmailError("failed to set sender", err)
	}

	// Set recipient
	if err := client.Rcpt(params.To); err != nil {
		return errors.NewEmailError("failed to set recipient", err)
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return errors.NewEmailError("failed to get data writer", err)
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			// Intentionally ignore close errors for cleanup
			_ = closeErr
		}
	}()

	if _, err := writer.Write([]byte(msg)); err != nil {
		return errors.NewEmailError("failed to write message", err)
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
