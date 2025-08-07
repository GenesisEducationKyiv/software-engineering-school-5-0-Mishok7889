package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"weatherapi.app/internal/ports"
)

type NotificationServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

type NotificationServiceConfig struct {
	Host string
	Port int
}

func NewNotificationServiceClient(config NotificationServiceConfig) (*NotificationServiceClient, error) {
	baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)

	return &NotificationServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *NotificationServiceClient) SendConfirmationEmail(ctx context.Context, email, city, confirmationURL string) error {
	req := map[string]interface{}{
		"email":            email,
		"confirmation_url": confirmationURL,
		"city":             city,
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal confirmation email request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/notifications/email/confirmation", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send HTTP request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *NotificationServiceClient) SendWelcomeEmail(ctx context.Context, email, city, frequency, unsubscribeURL string) error {
	req := map[string]interface{}{
		"email":           email,
		"city":            city,
		"frequency":       frequency,
		"unsubscribe_url": unsubscribeURL,
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal welcome email request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/notifications/email/welcome", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send HTTP request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *NotificationServiceClient) SendUnsubscribeConfirmationEmail(ctx context.Context, email, city string) error {
	req := map[string]interface{}{
		"email": email,
		"city":  city,
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal unsubscribe email request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/notifications/email/unsubscribe", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send HTTP request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}

// Implement remaining NotificationService interface methods
func (c *NotificationServiceClient) SendWeatherUpdates(ctx context.Context, frequency string) error {
	return fmt.Errorf("SendWeatherUpdates not implemented in subscription service client")
}

func (c *NotificationServiceClient) GetNotificationStats(ctx context.Context) (ports.NotificationStats, error) {
	return ports.NotificationStats{}, fmt.Errorf("GetNotificationStats not implemented in subscription service client")
}
