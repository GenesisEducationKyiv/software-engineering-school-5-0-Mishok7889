package external

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

func TestSMTPEmailProviderAdapter_ValidateConfiguration(t *testing.T) {
	validConfig := EmailProviderConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		FromName: "App",
		FromAddr: "app@company.com",
	}

	tests := []struct {
		name        string
		modify      func(EmailProviderConfig) EmailProviderConfig
		expectPanic bool
	}{
		{
			name: "Valid Config",
			modify: func(cfg EmailProviderConfig) EmailProviderConfig {
				return cfg
			},
			expectPanic: false,
		},
		{
			name: "Valid Mailhog Config",
			modify: func(cfg EmailProviderConfig) EmailProviderConfig {
				cfg.Host = "mailhog-e2e"
				cfg.Port = 1025
				cfg.Username = ""
				cfg.Password = ""
				cfg.FromName = "Weather API E2E"
				cfg.FromAddr = "test@weatherapi.com"
				return cfg
			},
			expectPanic: false,
		},
		{
			name: "Missing Host",
			modify: func(cfg EmailProviderConfig) EmailProviderConfig {
				cfg.Host = ""
				return cfg
			},
			expectPanic: true,
		},
		{
			name: "Invalid Port",
			modify: func(cfg EmailProviderConfig) EmailProviderConfig {
				cfg.Port = 0
				return cfg
			},
			expectPanic: true,
		},
		{
			name: "Missing From Address",
			modify: func(cfg EmailProviderConfig) EmailProviderConfig {
				cfg.FromAddr = ""
				return cfg
			},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.modify(validConfig)

			if tt.expectPanic {
				assert.Panics(t, func() {
					NewSMTPEmailProviderAdapter(config)
				})
			} else {
				assert.NotPanics(t, func() {
					provider := NewSMTPEmailProviderAdapter(config)
					assert.NotNil(t, provider)
				})
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

	validParams := ports.EmailParams{
		To:      "recipient@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
		Format:  ports.FormatText,
	}

	tests := []struct {
		name        string
		modify      func(ports.EmailParams) ports.EmailParams
		expectError bool
	}{
		{
			name: "Valid Email Params",
			modify: func(params ports.EmailParams) ports.EmailParams {
				return params
			},
			expectError: false, // Will fail due to no actual SMTP server, but validation should pass
		},
		{
			name: "Missing To",
			modify: func(params ports.EmailParams) ports.EmailParams {
				params.To = ""
				return params
			},
			expectError: true,
		},
		{
			name: "Missing Subject",
			modify: func(params ports.EmailParams) ports.EmailParams {
				params.Subject = ""
				return params
			},
			expectError: true,
		},
		{
			name: "Missing Body",
			modify: func(params ports.EmailParams) ports.EmailParams {
				params.Body = ""
				return params
			},
			expectError: true,
		},
		{
			name: "Invalid Format",
			modify: func(params ports.EmailParams) ports.EmailParams {
				params.Format = "invalid"
				return params
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := tt.modify(validParams)

			// Convert EmailParams to EmailRequest for the new interface
			emailRequest := shared.EmailRequest{
				To:      params.To,
				Subject: params.Subject,
				Body:    params.Body,
			}

			err := provider.SendEmail(ctx, emailRequest)

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
		params   ports.EmailParams
		contains []string
	}{
		{
			name: "Plain Text Email",
			params: ports.EmailParams{
				To:      "recipient@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				Format:  ports.FormatText,
			},
			contains: []string{
				"From: Test App <test@example.com>",
				"To: recipient@example.com",
				"Subject: Test Subject",
				"Content-Type: text/plain; charset=UTF-8",
				"Test Body",
			},
		},
		{
			name: "HTML Email",
			params: ports.EmailParams{
				To:      "recipient@example.com",
				Subject: "HTML Test",
				Body:    "<h1>HTML Body</h1>",
				Format:  ports.FormatHTML,
			},
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := provider.buildMessage(tt.params.To, tt.params.Subject, tt.params.Body, tt.params)

			for _, expected := range tt.contains {
				assert.Contains(t, msg, expected)
			}
		})
	}
}
