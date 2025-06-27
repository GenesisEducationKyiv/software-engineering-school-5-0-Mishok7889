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

// ConfirmationEmailParams holds parameters for sending confirmation emails
type ConfirmationEmailParams struct {
	Email      string
	ConfirmURL string
	City       string
}

// validateConfirmationEmailParams validates parameters for confirmation email
func (s *EmailService) validateConfirmationEmailParams(params ConfirmationEmailParams) error {
	if params.Email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if params.ConfirmURL == "" {
		return errors.NewValidationError("confirmation URL cannot be empty")
	}
	if params.City == "" {
		return errors.NewValidationError("city cannot be empty")
	}
	return nil
}

// WelcomeEmailParams holds parameters for sending welcome emails
type WelcomeEmailParams struct {
	Email          string
	City           string
	Frequency      string
	UnsubscribeURL string
}

// validateWelcomeEmailParams validates parameters for welcome email
func (s *EmailService) validateWelcomeEmailParams(params WelcomeEmailParams) error {
	if params.Email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if params.City == "" {
		return errors.NewValidationError("city cannot be empty")
	}
	if params.Frequency == "" {
		return errors.NewValidationError("frequency cannot be empty")
	}
	if params.UnsubscribeURL == "" {
		return errors.NewValidationError("unsubscribe URL cannot be empty")
	}
	return nil
}

// UnsubscribeEmailParams holds parameters for unsubscribe confirmation emails
type UnsubscribeEmailParams struct {
	Email string
	City  string
}

// validateUnsubscribeEmailParams validates parameters for unsubscribe email
func (s *EmailService) validateUnsubscribeEmailParams(params UnsubscribeEmailParams) error {
	if params.Email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if params.City == "" {
		return errors.NewValidationError("city cannot be empty")
	}
	return nil
}

// WeatherUpdateEmailParams holds parameters for weather update emails
type WeatherUpdateEmailParams struct {
	Email          string
	City           string
	Weather        *models.WeatherResponse
	UnsubscribeURL string
}

// validateWeatherUpdateEmailParams validates parameters for weather update email
func (s *EmailService) validateWeatherUpdateEmailParams(params WeatherUpdateEmailParams) error {
	if params.Email == "" {
		return errors.NewValidationError("email cannot be empty")
	}
	if params.City == "" {
		return errors.NewValidationError("city cannot be empty")
	}
	if params.Weather == nil {
		return errors.NewValidationError("weather data cannot be nil")
	}
	if params.UnsubscribeURL == "" {
		return errors.NewValidationError("unsubscribe URL cannot be empty")
	}
	return nil
}

// SendConfirmationEmail sends an email with a confirmation link
func (s *EmailService) SendConfirmationEmail(email, confirmURL, city string) error {
	slog.Debug("Sending confirmation email", "email", email, "city", city)

	params := ConfirmationEmailParams{
		Email:      email,
		ConfirmURL: confirmURL,
		City:       city,
	}

	if err := s.validateConfirmationEmailParams(params); err != nil {
		return err
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

// SendConfirmationEmailWithParams sends a confirmation email using parameter struct
func (s *EmailService) SendConfirmationEmailWithParams(params ConfirmationEmailParams) error {
	slog.Debug("Sending confirmation email", "email", params.Email, "city", params.City)

	if err := s.validateConfirmationEmailParams(params); err != nil {
		return err
	}

	subject := fmt.Sprintf("Confirm your weather subscription for %s", params.City)
	htmlContent := fmt.Sprintf(
		"<p>Please confirm your subscription to weather updates for %s by clicking the following link:</p>"+
			"<p><a href=\"%s\">Confirm Subscription</a></p>"+
			"<p>This link will expire in 24 hours.</p>",
		params.City, params.ConfirmURL,
	)

	return s.provider.SendEmail(params.Email, subject, htmlContent, true)
}

// SendWelcomeEmail sends a welcome email after subscription confirmation
func (s *EmailService) SendWelcomeEmail(email, city, frequency, unsubscribeURL string) error {
	slog.Debug("Sending welcome email", "email", email, "city", city, "frequency", frequency)

	params := WelcomeEmailParams{
		Email:          email,
		City:           city,
		Frequency:      frequency,
		UnsubscribeURL: unsubscribeURL,
	}

	if err := s.validateWelcomeEmailParams(params); err != nil {
		return err
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

// SendWelcomeEmailWithParams sends a welcome email using parameter struct
func (s *EmailService) SendWelcomeEmailWithParams(params WelcomeEmailParams) error {
	slog.Debug("Sending welcome email", "email", params.Email, "city", params.City, "frequency", params.Frequency)

	if err := s.validateWelcomeEmailParams(params); err != nil {
		return err
	}

	subject := fmt.Sprintf("Welcome to Weather Updates for %s", params.City)
	frequencyText := "every hour"
	if params.Frequency == "daily" {
		frequencyText = "every day"
	}

	htmlContent := fmt.Sprintf(
		"<p>Thank you for subscribing to %s weather updates for %s.</p>"+
			"<p>You will receive updates %s.</p>"+
			"<p>To unsubscribe, <a href=\"%s\">click here</a>.</p>",
		params.Frequency, params.City, frequencyText, params.UnsubscribeURL,
	)

	return s.provider.SendEmail(params.Email, subject, htmlContent, true)
}

// SendUnsubscribeConfirmationEmail sends a confirmation after unsubscribing
func (s *EmailService) SendUnsubscribeConfirmationEmail(email, city string) error {
	slog.Debug("Sending unsubscribe confirmation email", "email", email, "city", city)

	params := UnsubscribeEmailParams{
		Email: email,
		City:  city,
	}

	if err := s.validateUnsubscribeEmailParams(params); err != nil {
		return err
	}

	subject := fmt.Sprintf("You have unsubscribed from weather updates for %s", city)
	htmlContent := fmt.Sprintf(
		"<p>You have successfully unsubscribed from weather updates for %s.</p>",
		city,
	)

	return s.provider.SendEmail(email, subject, htmlContent, true)
}

// SendUnsubscribeConfirmationEmailWithParams sends unsubscribe confirmation using parameter struct
func (s *EmailService) SendUnsubscribeConfirmationEmailWithParams(params UnsubscribeEmailParams) error {
	slog.Debug("Sending unsubscribe confirmation email", "email", params.Email, "city", params.City)

	if err := s.validateUnsubscribeEmailParams(params); err != nil {
		return err
	}

	subject := fmt.Sprintf("You have unsubscribed from weather updates for %s", params.City)
	htmlContent := fmt.Sprintf(
		"<p>You have successfully unsubscribed from weather updates for %s.</p>",
		params.City,
	)

	return s.provider.SendEmail(params.Email, subject, htmlContent, true)
}

// SendWeatherUpdateEmail sends a weather update email to a subscriber
func (s *EmailService) SendWeatherUpdateEmail(email, city string, weather *models.WeatherResponse, unsubscribeURL string) error {
	slog.Debug("Sending weather update email", "email", email, "city", city, "temp", weather.Temperature)

	params := WeatherUpdateEmailParams{
		Email:          email,
		City:           city,
		Weather:        weather,
		UnsubscribeURL: unsubscribeURL,
	}

	if err := s.validateWeatherUpdateEmailParams(params); err != nil {
		return err
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

// SendWeatherUpdateEmailWithParams sends weather update email using parameter struct
func (s *EmailService) SendWeatherUpdateEmailWithParams(params WeatherUpdateEmailParams) error {
	slog.Debug("Sending weather update email", "email", params.Email, "city", params.City, "temp", params.Weather.Temperature)

	if err := s.validateWeatherUpdateEmailParams(params); err != nil {
		return err
	}

	subject := fmt.Sprintf("Weather Update for %s", params.City)
	htmlContent := fmt.Sprintf(
		"<h2>Current weather for %s</h2>"+
			"<p><strong>Temperature:</strong> %.1f°C</p>"+
			"<p><strong>Humidity:</strong> %.1f%%</p>"+
			"<p><strong>Description:</strong> %s</p>"+
			"<p>To unsubscribe, <a href=\"%s\">click here</a>.</p>",
		params.City, params.Weather.Temperature, params.Weather.Humidity, params.Weather.Description, params.UnsubscribeURL,
	)

	return s.provider.SendEmail(params.Email, subject, htmlContent, true)
}
