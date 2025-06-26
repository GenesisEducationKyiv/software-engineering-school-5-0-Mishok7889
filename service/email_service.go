package service

import (
	"fmt"
	"log/slog"

	"weatherapi.app/errors"
	"weatherapi.app/models"
	"weatherapi.app/providers"
)

// EmailService handles email operations using a provider
type EmailService struct {
	provider providers.EmailProvider
}

// NewEmailService creates a new email service with the specified provider
func NewEmailService(provider providers.EmailProvider) *EmailService {
	return &EmailService{
		provider: provider,
	}
}

// SendConfirmationEmail sends an email with a confirmation link
func (s *EmailService) SendConfirmationEmail(email, confirmURL, city string) error {
	slog.Debug("Sending confirmation email", "email", email, "city", city)

	if email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if confirmURL == "" {
		return errors.NewValidationError("confirmation URL cannot be empty")
	}
	if city == "" {
		return errors.NewValidationError("city cannot be empty")
	}

	subject := fmt.Sprintf("Confirm your weather subscription for %s", city)
	htmlContent := fmt.Sprintf(
		"<p>Please confirm your subscription to weather updates for %s by clicking the following link:</p>"+
			"<p><a href=\"%s\">Confirm Subscription</a></p>"+
			"<p>This link will expire in 24 hours.</p>",
		city, confirmURL,
	)

	return s.provider.SendEmail(email, subject, htmlContent, true)
}

// SendWelcomeEmail sends a welcome email after subscription confirmation
func (s *EmailService) SendWelcomeEmail(email, city, frequency, unsubscribeURL string) error {
	slog.Debug("Sending welcome email", "email", email, "city", city, "frequency", frequency)

	if email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if city == "" {
		return errors.NewValidationError("city cannot be empty")
	}
	if frequency == "" {
		return errors.NewValidationError("frequency cannot be empty")
	}
	if unsubscribeURL == "" {
		return errors.NewValidationError("unsubscribe URL cannot be empty")
	}

	subject := fmt.Sprintf("Welcome to Weather Updates for %s", city)
	frequencyText := "every hour"
	if frequency == "daily" {
		frequencyText = "every day"
	}

	htmlContent := fmt.Sprintf(
		"<p>Thank you for subscribing to %s weather updates for %s.</p>"+
			"<p>You will receive updates %s.</p>"+
			"<p>To unsubscribe, <a href=\"%s\">click here</a>.</p>",
		frequency, city, frequencyText, unsubscribeURL,
	)

	return s.provider.SendEmail(email, subject, htmlContent, true)
}

// SendUnsubscribeConfirmationEmail sends a confirmation after unsubscribing
func (s *EmailService) SendUnsubscribeConfirmationEmail(email, city string) error {
	slog.Debug("Sending unsubscribe confirmation email", "email", email, "city", city)

	if email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if city == "" {
		return errors.NewValidationError("city cannot be empty")
	}

	subject := fmt.Sprintf("You have unsubscribed from weather updates for %s", city)
	htmlContent := fmt.Sprintf(
		"<p>You have successfully unsubscribed from weather updates for %s.</p>",
		city,
	)

	return s.provider.SendEmail(email, subject, htmlContent, true)
}

// SendWeatherUpdateEmail sends a weather update email to a subscriber
func (s *EmailService) SendWeatherUpdateEmail(email, city string, weather *models.WeatherResponse, unsubscribeURL string) error {
	slog.Debug("Sending weather update email", "email", email, "city", city, "temp", weather.Temperature)

	if email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if city == "" {
		return errors.NewValidationError("city cannot be empty")
	}
	if weather == nil {
		return errors.NewValidationError("weather data cannot be nil")
	}
	if unsubscribeURL == "" {
		return errors.NewValidationError("unsubscribe URL cannot be empty")
	}

	subject := fmt.Sprintf("Weather Update for %s", city)
	htmlContent := fmt.Sprintf(
		"<h2>Current weather for %s</h2>"+
			"<p><strong>Temperature:</strong> %.1f°C</p>"+
			"<p><strong>Humidity:</strong> %.1f%%</p>"+
			"<p><strong>Description:</strong> %s</p>"+
			"<p>To unsubscribe, <a href=\"%s\">click here</a>.</p>",
		city, weather.Temperature, weather.Humidity, weather.Description, unsubscribeURL,
	)

	return s.provider.SendEmail(email, subject, htmlContent, true)
}
