package providers

import (
	"fmt"
	"net/smtp"
	"strings"

	"weatherapi.app/config"
	"weatherapi.app/errors"
)

// SMTPEmailProvider implements EmailProvider using SMTP
type SMTPEmailProvider struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromName     string
	fromAddress  string
}

// NewSMTPEmailProvider creates a new SMTP email provider
func NewSMTPEmailProvider(config *config.EmailConfig) *SMTPEmailProvider {
	return &SMTPEmailProvider{
		smtpHost:     config.SMTPHost,
		smtpPort:     config.SMTPPort,
		smtpUsername: config.SMTPUsername,
		smtpPassword: config.SMTPPassword,
		fromName:     config.FromName,
		fromAddress:  config.FromAddress,
	}
}

// validateSendEmailParams validates the input parameters for sending an email
func (p *SMTPEmailProvider) validateSendEmailParams(to, subject string) error {
	if to == "" {
		return errors.NewValidationError("recipient email cannot be empty")
	}
	if subject == "" {
		return errors.NewValidationError("email subject cannot be empty")
	}
	return nil
}

// SendEmail sends an email using SMTP
func (p *SMTPEmailProvider) SendEmail(to, subject, body string, isHTML bool) error {
	if err := p.validateSendEmailParams(to, subject); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", p.smtpUsername, p.smtpPassword, p.smtpHost)

	mimeHeaders := "MIME-Version: 1.0\r\n"
	contentType := "Content-Type: text/plain; charset=UTF-8\r\n"
	if isHTML {
		contentType = "Content-Type: text/html; charset=UTF-8\r\n"
	}

	// Remove line breaks from subject to prevent header injection
	subject = strings.ReplaceAll(strings.ReplaceAll(subject, "\r\n", ""), "\n", "")

	from := fmt.Sprintf("%s <%s>", p.fromName, p.fromAddress)
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n%s%s\r\n",
		from, to, subject, mimeHeaders, contentType)

	message := headers + body
	smtpAddr := fmt.Sprintf("%s:%d", p.smtpHost, p.smtpPort)

	err := smtp.SendMail(smtpAddr, auth, p.fromAddress, []string{to}, []byte(message))
	if err != nil {
		return errors.NewEmailError("failed to send email", err)
	}

	return nil
}
