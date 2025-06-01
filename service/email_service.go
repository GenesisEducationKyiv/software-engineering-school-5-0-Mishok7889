package service

import (
	"fmt"
	"net/smtp"
	"strings"

	"weatherapi.app/config"
	"weatherapi.app/models"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(config *config.Config) *EmailService {
	return &EmailService{
		config: config,
	}
}

// sendEmail sends an email using Gmail SMTP server
func (s *EmailService) sendEmail(to, subject, body string, isHtml bool) error {
	fmt.Printf("[DEBUG] EmailService.sendEmail called with: to=%s, subject=%s\n", to, subject)

	// SMTP server configuration
	smtpHost := s.config.Email.SMTPHost
	smtpPort := s.config.Email.SMTPPort
	smtpUsername := s.config.Email.SMTPUsername
	smtpPassword := s.config.Email.SMTPPassword
	fromName := s.config.Email.FromName
	fromAddress := s.config.Email.FromAddress

	// Set up authentication information
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)

	// Set up email headers
	mimeHeaders := "MIME-Version: 1.0\r\n"
	contentType := "Content-Type: text/plain; charset=UTF-8\r\n"
	if isHtml {
		contentType = "Content-Type: text/html; charset=UTF-8\r\n"
	}

	subject = strings.ReplaceAll(subject, "\r\n", "")
	subject = strings.ReplaceAll(subject, "\n", "")
	
	from := fmt.Sprintf("%s <%s>", fromName, fromAddress)
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n%s%s\r\n", 
		from, to, subject, mimeHeaders, contentType)

	// Combine headers and message body
	message := headers + body

	// Connect to the SMTP server and send email
	smtpAddr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	fmt.Printf("[DEBUG] Sending email via SMTP: server=%s, from=%s, to=%s\n", smtpAddr, fromAddress, to)
	err := smtp.SendMail(smtpAddr, auth, fromAddress, []string{to}, []byte(message))
	if err != nil {
		fmt.Printf("[ERROR] Failed to send email: %v\n", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("[DEBUG] Email sent successfully")
	return nil
}

func (s *EmailService) SendConfirmationEmail(email, confirmURL, city string) error {
	fmt.Printf("[DEBUG] SendConfirmationEmail called for: %s, city: %s\n", email, city)

	subject := fmt.Sprintf("Confirm your weather subscription for %s", city)

	htmlContent := fmt.Sprintf(
		"<p>Please confirm your subscription to weather updates for %s by clicking the following link:</p>"+
			"<p><a href=\"%s\">Confirm Subscription</a></p>"+
			"<p>This link will expire in 24 hours.</p>",
		city, confirmURL,
	)

	return s.sendEmail(email, subject, htmlContent, true)
}

func (s *EmailService) SendWelcomeEmail(email, city, frequency, unsubscribeURL string) error {
	fmt.Printf("[DEBUG] SendWelcomeEmail called for: %s, city: %s, frequency: %s\n",
		email, city, frequency)

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

	return s.sendEmail(email, subject, htmlContent, true)
}

func (s *EmailService) SendUnsubscribeConfirmationEmail(email, city string) error {
	fmt.Printf("[DEBUG] SendUnsubscribeConfirmationEmail called for: %s, city: %s\n", email, city)

	subject := fmt.Sprintf("You have unsubscribed from weather updates for %s", city)

	htmlContent := fmt.Sprintf(
		"<p>You have successfully unsubscribed from weather updates for %s.</p>",
		city,
	)

	return s.sendEmail(email, subject, htmlContent, true)
}

func (s *EmailService) SendWeatherUpdateEmail(email, city string, weather *models.WeatherResponse, unsubscribeURL string) error {
	fmt.Printf("[DEBUG] SendWeatherUpdateEmail called for: %s, city: %s\n", email, city)

	subject := fmt.Sprintf("Weather Update for %s", city)

	htmlContent := fmt.Sprintf(
		"<h2>Current weather for %s</h2>"+
			"<p><strong>Temperature:</strong> %.1f°C</p>"+
			"<p><strong>Humidity:</strong> %.1f%%</p>"+
			"<p><strong>Description:</strong> %s</p>"+
			"<p>To unsubscribe, <a href=\"%s\">click here</a>.</p>",
		city, weather.Temperature, weather.Humidity, weather.Description, unsubscribeURL,
	)

	return s.sendEmail(email, subject, htmlContent, true)
}
