package messaging

import (
	"context"
	"fmt"
	"net/smtp"

	"weatherapi.app/internal/core/shared"
)

type EmailProviderAdapter struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromName     string
	fromAddress  string
}

type EmailProviderConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromName     string
	FromAddress  string
}

func NewEmailProviderAdapter(config EmailProviderConfig) (*EmailProviderAdapter, error) {
	return &EmailProviderAdapter{
		smtpHost:     config.SMTPHost,
		smtpPort:     config.SMTPPort,
		smtpUsername: config.SMTPUsername,
		smtpPassword: config.SMTPPassword,
		fromName:     config.FromName,
		fromAddress:  config.FromAddress,
	}, nil
}

func (e *EmailProviderAdapter) SendEmail(ctx context.Context, req shared.EmailRequest) error {
	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", req.To, req.Subject, req.Body))

	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	if err := smtp.SendMail(addr, auth, e.fromAddress, []string{req.To}, msg); err != nil {
		return fmt.Errorf("send email to %s: %w", req.To, err)
	}

	return nil
}
