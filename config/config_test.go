package config

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	apperrors "weatherapi.app/errors"
)

func TestEmailConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      EmailConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config with credentials",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "user@gmail.com",
				SMTPPassword: "password",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: false,
		},
		{
			name: "Valid config without credentials (MailHog)",
			config: EmailConfig{
				SMTPHost:     "mailhog",
				SMTPPort:     1025,
				SMTPUsername: "",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: false,
		},
		{
			name: "Invalid - only username provided",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "user@gmail.com",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: true,
			errorMsg:    "EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD must both be provided or both be empty",
		},
		{
			name: "Invalid - only password provided",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "",
				SMTPPassword: "password",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: true,
			errorMsg:    "EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD must both be provided or both be empty",
		},
		{
			name: "Invalid - empty host",
			config: EmailConfig{
				SMTPHost:     "",
				SMTPPort:     587,
				SMTPUsername: "",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: true,
			errorMsg:    "EMAIL_SMTP_HOST cannot be empty",
		},
		{
			name: "Invalid - invalid from address",
			config: EmailConfig{
				SMTPHost:     "mailhog",
				SMTPPort:     1025,
				SMTPUsername: "",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "invalid-email",
			},
			expectError: true,
			errorMsg:    "EMAIL_FROM_ADDRESS must be a valid email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var appErr *apperrors.AppError
				assert.True(t, errors.As(err, &appErr))
				assert.Equal(t, apperrors.ConfigurationError, appErr.Type)
				assert.Contains(t, appErr.Message, tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_LoadWithMailHogCredentials(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			if len(env) > 0 {
				parts := []string{"", ""}
				for i, part := range []rune(env) {
					if part == '=' {
						parts = []string{string([]rune(env)[:i]), string([]rune(env)[i+1:])}
						break
					}
				}
				if len(parts) == 2 && parts[0] != "" {
					_ = os.Setenv(parts[0], parts[1]) // Ignore error in cleanup
				}
			}
		}
	}()

	// Clear environment and set minimal required values for MailHog testing
	os.Clearenv()
	assert.NoError(t, os.Setenv("WEATHER_API_KEY", "test-api-key"))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_HOST", "mailhog"))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_PORT", "1025"))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_USERNAME", ""))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_PASSWORD", ""))
	assert.NoError(t, os.Setenv("EMAIL_FROM_ADDRESS", "test@example.com"))
	assert.NoError(t, os.Setenv("DB_HOST", "localhost"))
	assert.NoError(t, os.Setenv("DB_USER", "testuser"))
	assert.NoError(t, os.Setenv("DB_NAME", "testdb"))

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "mailhog", config.Email.SMTPHost)
	assert.Equal(t, 1025, config.Email.SMTPPort)
	assert.Equal(t, "", config.Email.SMTPUsername)
	assert.Equal(t, "", config.Email.SMTPPassword)
}
