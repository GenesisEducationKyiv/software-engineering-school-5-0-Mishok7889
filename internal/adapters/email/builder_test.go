package email

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
)

const (
	testBaseURL           = "http://localhost:8080"
	expectedTemplateCount = 3 // confirmation, welcome, unsubscribe
)

func TestNewBuilder(t *testing.T) {
	mockConfig := &mocks.ConfigProvider{}
	mockConfig.On("GetAppConfig").Return(ports.AppConfig{
		BaseURL: testBaseURL,
	})

	builder, err := NewBuilder(mockConfig)

	assert.NoError(t, err)
	assert.NotNil(t, builder)
	assert.Len(t, builder.templates, expectedTemplateCount)
}

func TestBuilder_BuildConfirmationEmail(t *testing.T) {
	mockConfig := &mocks.ConfigProvider{}
	mockConfig.On("GetAppConfig").Return(ports.AppConfig{
		BaseURL: testBaseURL,
	})

	builder, err := NewBuilder(mockConfig)
	assert.NoError(t, err)

	// Pass full URL as the subscription service would do in microservices architecture
	fullConfirmationURL := testBaseURL + "/api/confirm/test-token-123"
	email, err := builder.BuildConfirmationEmail("New York", fullConfirmationURL)

	assert.NoError(t, err)
	assert.Equal(t, subscription.EmailSubjectConfirmation, email.Subject)
	assert.Equal(t, ports.FormatHTML, email.Format)
	assert.Contains(t, email.Body, "New York")
	assert.Contains(t, email.Body, fullConfirmationURL)
	assert.Contains(t, email.Body, "Confirm Your Weather Subscription")
}

func TestBuilder_BuildWelcomeEmail(t *testing.T) {
	mockConfig := &mocks.ConfigProvider{}
	mockConfig.On("GetAppConfig").Return(ports.AppConfig{
		BaseURL: testBaseURL,
	})

	builder, err := NewBuilder(mockConfig)
	assert.NoError(t, err)

	// Pass full URL as the subscription service would do in microservices architecture
	fullUnsubscribeURL := testBaseURL + "/api/unsubscribe/unsubscribe-token-456"
	email, err := builder.BuildWelcomeEmail("London", "daily", fullUnsubscribeURL)

	assert.NoError(t, err)
	assert.Equal(t, subscription.EmailSubjectWelcome, email.Subject)
	assert.Equal(t, ports.FormatHTML, email.Format)
	assert.Contains(t, email.Body, "London")
	assert.Contains(t, email.Body, "daily")
	assert.Contains(t, email.Body, fullUnsubscribeURL)
	assert.Contains(t, email.Body, "Welcome to Weather Updates!")
}

func TestBuilder_BuildUnsubscribeEmail(t *testing.T) {
	mockConfig := &mocks.ConfigProvider{}
	mockConfig.On("GetAppConfig").Return(ports.AppConfig{
		BaseURL: testBaseURL,
	})

	builder, err := NewBuilder(mockConfig)
	assert.NoError(t, err)

	email, err := builder.BuildUnsubscribeEmail("Paris")

	assert.NoError(t, err)
	assert.Equal(t, subscription.EmailSubjectUnsubscribe, email.Subject)
	assert.Equal(t, ports.FormatHTML, email.Format)
	assert.Contains(t, email.Body, "Paris")
	assert.Contains(t, email.Body, "Unsubscribed Successfully")
}

func TestBuilder_InvalidTemplate(t *testing.T) {
	builder := &Builder{
		templates: make(map[string]*template.Template),
	}

	_, err := builder.executeTemplate("nonexistent", EmailData{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template nonexistent not found")
}
