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
	addr     string // Pre-computed server address
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
	provider := &SMTPEmailProviderAdapter{
		host:     config.Host,
		port:     config.Port,
		username: config.Username,
		password: config.Password,
		fromName: config.FromName,
		fromAddr: config.FromAddr,
		addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
	}

	if err := provider.ValidateConfiguration(); err != nil {
		panic(fmt.Sprintf("invalid email provider configuration: %v", err))
	}

	return provider
}

// SendEmail sends an email using SMTP with flexible authentication and TLS
func (p *SMTPEmailProviderAdapter) SendEmail(ctx context.Context, params ports.EmailParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	client, err := p.establishConnection()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if err := p.authenticateConnection(client); err != nil {
		return err
	}

	return p.sendMessage(client, params)
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

// establishConnection creates and configures the SMTP connection
func (p *SMTPEmailProviderAdapter) establishConnection() (*smtp.Client, error) {
	client, err := smtp.Dial(p.addr)
	if err != nil {
		return nil, infrastructure.NewEmailError("failed to connect to SMTP server", err)
	}

	if err := p.startTLS(client); err != nil {
		return nil, err
	}

	return client, nil
}

// startTLS establishes a secure TLS connection if supported by the server
func (p *SMTPEmailProviderAdapter) startTLS(client *smtp.Client) error {
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: p.host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return infrastructure.NewEmailError("failed to establish secure TLS connection", err)
		}
	}
	return nil
}

// authenticateConnection handles SMTP authentication
func (p *SMTPEmailProviderAdapter) authenticateConnection(client *smtp.Client) error {
	if p.username != "" && p.password != "" {
		auth := smtp.PlainAuth("", p.username, p.password, p.host)
		if err := client.Auth(auth); err != nil {
			return infrastructure.NewEmailError("failed to authenticate", err)
		}
	}
	return nil
}

// sendMessage sends the email message through the SMTP client
func (p *SMTPEmailProviderAdapter) sendMessage(client *smtp.Client, params ports.EmailParams) error {
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

	msg := p.buildMessage(params.To, params.Subject, params.Body, params)
	if _, err := writer.Write([]byte(msg)); err != nil {
		return infrastructure.NewEmailError("failed to write message", err)
	}

	return nil
}

// buildMessage constructs the email message
func (p *SMTPEmailProviderAdapter) buildMessage(to, subject, body string, params ports.EmailParams) string {
	from := fmt.Sprintf("%s <%s>", p.fromName, p.fromAddr)
	contentType := "text/plain"
	if params.IsHTML() {
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
