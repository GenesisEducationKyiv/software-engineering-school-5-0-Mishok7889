package external

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/ports"
)

func TestSMTPEmailProviderAdapter_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      EmailProviderConfig
		expectError bool
	}{
		{
			name: "Valid Mailhog Config",
			config: EmailProviderConfig{
				Host:     "mailhog-e2e",
				Port:     1025,
				Username: "",
				Password: "",
				FromName: "Weather API E2E",
				FromAddr: "test@weatherapi.com",
			},
			expectError: false,
		},
		{
			name: "Valid Production Config",
			config: EmailProviderConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user@gmail.com",
				Password: "password123",
				FromName: "Weather API",
				FromAddr: "noreply@weatherapi.com",
			},
			expectError: false,
		},
		{
			name: "Missing Host",
			config: EmailProviderConfig{
				Host:     "",
				Port:     587,
				Username: "user",
				Password: "pass",
				FromName: "App",
				FromAddr: "app@company.com",
			},
			expectError: true,
		},
		{
			name: "Invalid Port",
			config: EmailProviderConfig{
				Host:     "smtp.example.com",
				Port:     0,
				Username: "user",
				Password: "pass",
				FromName: "App",
				FromAddr: "app@company.com",
			},
			expectError: true,
		},
		{
			name: "Missing From Address",
			config: EmailProviderConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user",
				Password: "pass",
				FromName: "App",
				FromAddr: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewSMTPEmailProviderAdapter(tt.config).(*SMTPEmailProviderAdapter)
			err := provider.ValidateConfiguration()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSMTPEmailProviderAdapter_SendEmailValidation(t *testing.T) {
	config := EmailProviderConfig{
		Host:     "mailhog-e2e",
		Port:     1025,
		Username: "",
		Password: "",
		FromName: "Test",
		FromAddr: "test@example.com",
	}

	provider := NewSMTPEmailProviderAdapter(config)
	ctx := context.Background()

	tests := []struct {
		name        string
		params      ports.EmailParams
		expectError bool
	}{
		{
			name: "Valid Email Params",
			params: ports.EmailParams{
				To:      "recipient@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				IsHTML:  false,
			},
			expectError: false, // Will fail due to no actual SMTP server, but validation should pass
		},
		{
			name: "Missing To",
			params: ports.EmailParams{
				To:      "",
				Subject: "Test Subject",
				Body:    "Test Body",
				IsHTML:  false,
			},
			expectError: true,
		},
		{
			name: "Missing Subject",
			params: ports.EmailParams{
				To:      "recipient@example.com",
				Subject: "",
				Body:    "Test Body",
				IsHTML:  false,
			},
			expectError: true,
		},
		{
			name: "Missing Body",
			params: ports.EmailParams{
				To:      "recipient@example.com",
				Subject: "Test Subject",
				Body:    "",
				IsHTML:  false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.SendEmail(ctx, tt.params)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// For the valid case, we expect either success OR a connection error
				// since we're not running an actual SMTP server
				// We're mainly testing that validation passes
				if err != nil {
					// If there's an error, it should be a connection error, not validation
					assert.Contains(t, err.Error(), "failed to connect")
				}
			}
		})
	}
}

func TestSMTPEmailProviderAdapter_BuildMessage(t *testing.T) {
	config := EmailProviderConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		FromName: "Test App",
		FromAddr: "test@example.com",
	}

	provider := NewSMTPEmailProviderAdapter(config).(*SMTPEmailProviderAdapter)

	tests := []struct {
		name     string
		from     string
		to       string
		subject  string
		body     string
		isHTML   bool
		contains []string
	}{
		{
			name:    "Plain Text Email",
			from:    "Test App <test@example.com>",
			to:      "recipient@example.com",
			subject: "Test Subject",
			body:    "Test Body",
			isHTML:  false,
			contains: []string{
				"From: Test App <test@example.com>",
				"To: recipient@example.com",
				"Subject: Test Subject",
				"Content-Type: text/plain; charset=UTF-8",
				"Test Body",
			},
		},
		{
			name:    "HTML Email",
			from:    "Test App <test@example.com>",
			to:      "recipient@example.com",
			subject: "HTML Test",
			body:    "<h1>HTML Body</h1>",
			isHTML:  true,
			contains: []string{
				"From: Test App <test@example.com>",
				"To: recipient@example.com",
				"Subject: HTML Test",
				"Content-Type: text/html; charset=UTF-8",
				"<h1>HTML Body</h1>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := provider.buildMessage(tt.from, tt.to, tt.subject, tt.body, tt.isHTML)

			for _, expected := range tt.contains {
				assert.Contains(t, msg, expected)
			}
		})
	}
}
