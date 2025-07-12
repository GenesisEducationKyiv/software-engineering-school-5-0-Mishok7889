package external

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
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
		return infrastructure.NewValidationError("recipient email cannot be empty")
	}
	if params.Subject == "" {
		return infrastructure.NewValidationError("email subject cannot be empty")
	}
	if params.Body == "" {
		return infrastructure.NewValidationError("email body cannot be empty")
	}

	from := fmt.Sprintf("%s <%s>", p.fromName, p.fromAddr)
	msg := p.buildMessage(from, params.To, params.Subject, params.Body, params.IsHTML)
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	client, err := smtp.Dial(addr)
	if err != nil {
		return infrastructure.NewEmailError("failed to connect to SMTP server", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: p.host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return infrastructure.NewEmailError("failed to establish secure TLS connection", err)
		}
	}

	if p.username != "" && p.password != "" {
		auth := smtp.PlainAuth("", p.username, p.password, p.host)
		if err := client.Auth(auth); err != nil {
			return infrastructure.NewEmailError("failed to authenticate", err)
		}
	}

	if err := client.Mail(p.fromAddr); err != nil {
		return infrastructure.NewEmailError("failed to set sender", err)
	}

	if err := client.Rcpt(params.To); err != nil {
		return infrastructure.NewEmailError("failed to set recipient", err)
	}

	writer, err := client.Data()
	if err != nil {
		return infrastructure.NewEmailError("failed to get data writer", err)
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if _, err := writer.Write([]byte(msg)); err != nil {
		return infrastructure.NewEmailError("failed to write message", err)
	}

	return nil
}

// ValidateConfiguration validates the email provider configuration
func (p *SMTPEmailProviderAdapter) ValidateConfiguration() error {
	if p.host == "" {
		return infrastructure.NewConfigurationError("SMTP host cannot be empty")
	}
	if p.port < 1 || p.port > 65535 {
		return infrastructure.NewConfigurationError("SMTP port must be between 1 and 65535")
	}
	if p.fromAddr == "" {
		return infrastructure.NewConfigurationError("from address cannot be empty")
	}
	if p.fromName == "" {
		return infrastructure.NewConfigurationError("from name cannot be empty")
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
