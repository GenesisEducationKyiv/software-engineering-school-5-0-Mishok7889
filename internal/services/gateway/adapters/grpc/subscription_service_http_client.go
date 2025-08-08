package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

type SubscriptionServiceHTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

type SubscriptionServiceHTTPConfig struct {
	Host string
	Port int
}

func NewSubscriptionServiceHTTPClient(config SubscriptionServiceHTTPConfig) (*SubscriptionServiceHTTPClient, error) {
	baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)

	return &SubscriptionServiceHTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// prepareJSONRequest creates an HTTP request with JSON content type header
func (c *SubscriptionServiceHTTPClient) prepareJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *SubscriptionServiceHTTPClient) Subscribe(ctx context.Context, email, city, frequency string) error {
	reqBody := map[string]string{
		"email":     email,
		"city":      city,
		"frequency": frequency,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := c.prepareJSONRequest(ctx, http.MethodPost, c.baseURL+"/api/v1/subscriptions", jsonBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("subscription service returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *SubscriptionServiceHTTPClient) GetConfirmedSubscriptions(ctx context.Context, frequency string) ([]*ports.SubscriptionServiceData, error) {
	return nil, fmt.Errorf("not implemented - would call subscription service API")
}

func (c *SubscriptionServiceHTTPClient) GetActiveSubscriptionsByCity(ctx context.Context, city string) ([]shared.Subscription, error) {
	return nil, fmt.Errorf("not implemented - would call subscription service API")
}

func (c *SubscriptionServiceHTTPClient) GetActiveSubscriptionsByFrequency(ctx context.Context, frequency shared.Frequency) ([]shared.Subscription, error) {
	return nil, fmt.Errorf("not implemented - would call subscription service API")
}

func (c *SubscriptionServiceHTTPClient) FindByID(ctx context.Context, id uint) (*ports.SubscriptionServiceData, error) {
	return nil, fmt.Errorf("not implemented - would call subscription service API")
}

func (c *SubscriptionServiceHTTPClient) ConfirmSubscription(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v1/confirm/"+token, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("subscription service returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *SubscriptionServiceHTTPClient) Unsubscribe(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v1/unsubscribe/"+token, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("subscription service returned status %d", resp.StatusCode)
	}

	return nil
}
