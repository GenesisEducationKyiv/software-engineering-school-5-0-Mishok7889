package email

import (
	"bytes"
	"fmt"
	"text/template"

	"weatherapi.app/internal/ports"
)

const (
	templateNameConfirmation = "confirmation"
	templateNameWelcome      = "welcome"
	templateNameUnsubscribe  = "unsubscribe"

	// Email subjects
	EmailSubjectConfirmation = "Confirm your weather subscription"
	EmailSubjectWelcome      = "Welcome to Weather Updates!"
	EmailSubjectUnsubscribe  = "You have been unsubscribed from weather updates"

	// Email body templates
	EmailBodyConfirmation = `
		<h2>Confirm Your Weather Subscription</h2>
		<p>Hello!</p>
		<p>Thank you for subscribing to weather updates for <strong>{{.City}}</strong>.</p>
		<p>Please click the link below to confirm your subscription:</p>
		<p><a href="{{.ConfirmURL}}">Confirm Subscription</a></p>
		<p>If you didn't request this subscription, you can safely ignore this email.</p>
	`

	EmailBodyWelcome = `
		<h2>Welcome to Weather Updates!</h2>
		<p>Hello!</p>
		<p>Your subscription for <strong>{{.City}}</strong> weather updates has been confirmed.</p>
		<p>You will receive <strong>{{.Frequency}}</strong> weather updates.</p>
		<p>If you wish to unsubscribe, click <a href="{{.UnsubscribeURL}}">here</a>.</p>
	`

	EmailBodyUnsubscribe = `
		<h2>Unsubscribed Successfully</h2>
		<p>Hello!</p>
		<p>You have been successfully unsubscribed from weather updates for <strong>{{.City}}</strong>.</p>
		<p>We're sorry to see you go!</p>
	`
)

// Builder handles email content generation using templates
type Builder struct {
	config    ports.ConfigProvider
	templates map[string]*template.Template
}

// Compile-time verification that Builder implements EmailBuilder
var _ ports.EmailBuilder = (*Builder)(nil)

// EmailData represents common data for all email templates
type EmailData struct {
	City           string
	Frequency      string
	ConfirmURL     string
	UnsubscribeURL string
	BaseURL        string
}

// NewBuilder creates a new email builder with predefined templates
func NewBuilder(config ports.ConfigProvider) (*Builder, error) {
	builder := &Builder{
		config:    config,
		templates: make(map[string]*template.Template),
	}

	if err := builder.loadTemplates(); err != nil {
		return nil, fmt.Errorf("load email templates: %w", err)
	}

	return builder, nil
}

// loadTemplates initializes all email templates
func (b *Builder) loadTemplates() error {
	templates := map[string]string{
		templateNameConfirmation: EmailBodyConfirmation,
		templateNameWelcome:      EmailBodyWelcome,
		templateNameUnsubscribe:  EmailBodyUnsubscribe,
	}

	for name, content := range templates {
		tmpl, err := template.New(name).Parse(content)
		if err != nil {
			return fmt.Errorf("parse template %s: %w", name, err)
		}
		b.templates[name] = tmpl
	}

	return nil
}

// BuildConfirmationEmail creates a confirmation email
func (b *Builder) BuildConfirmationEmail(city, confirmationURL string) (ports.EmailParams, error) {
	data := EmailData{
		City:       city,
		ConfirmURL: confirmationURL,
		BaseURL:    b.config.GetAppConfig().BaseURL,
	}

	body, err := b.executeTemplate(templateNameConfirmation, data)
	if err != nil {
		return ports.EmailParams{}, fmt.Errorf("execute confirmation template: %w", err)
	}

	return ports.EmailParams{
		Subject: EmailSubjectConfirmation,
		Body:    body,
		Format:  ports.FormatHTML,
	}, nil
}

// BuildWelcomeEmail creates a welcome email
func (b *Builder) BuildWelcomeEmail(city, frequency, unsubscribeURL string) (ports.EmailParams, error) {
	data := EmailData{
		City:           city,
		Frequency:      frequency,
		UnsubscribeURL: unsubscribeURL,
		BaseURL:        b.config.GetAppConfig().BaseURL,
	}

	body, err := b.executeTemplate(templateNameWelcome, data)
	if err != nil {
		return ports.EmailParams{}, fmt.Errorf("execute welcome template: %w", err)
	}

	return ports.EmailParams{
		Subject: EmailSubjectWelcome,
		Body:    body,
		Format:  ports.FormatHTML,
	}, nil
}

// BuildUnsubscribeEmail creates an unsubscribe confirmation email
func (b *Builder) BuildUnsubscribeEmail(city string) (ports.EmailParams, error) {
	data := EmailData{
		City:    city,
		BaseURL: b.config.GetAppConfig().BaseURL,
	}

	body, err := b.executeTemplate(templateNameUnsubscribe, data)
	if err != nil {
		return ports.EmailParams{}, fmt.Errorf("execute unsubscribe template: %w", err)
	}

	return ports.EmailParams{
		Subject: EmailSubjectUnsubscribe,
		Body:    body,
		Format:  ports.FormatHTML,
	}, nil
}

// executeTemplate executes a template with the given data
func (b *Builder) executeTemplate(templateName string, data EmailData) (string, error) {
	tmpl, exists := b.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
