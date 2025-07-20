package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

// HTTPWeatherClient handles common HTTP operations for weather providers
type HTTPWeatherClient struct {
	client HTTPClient
	logger ports.Logger
}

// HTTPWeatherRequest represents a weather API request
type HTTPWeatherRequest struct {
	URL          string
	ProviderName string
}

// HTTPWeatherResponse represents a weather API response
type HTTPWeatherResponse struct {
	Body       io.ReadCloser
	StatusCode int
}

// NewHTTPWeatherClient creates a new HTTP weather client
func NewHTTPWeatherClient(client HTTPClient, logger ports.Logger) *HTTPWeatherClient {
	return &HTTPWeatherClient{
		client: client,
		logger: logger,
	}
}

// ExecuteRequest executes an HTTP request and handles common error scenarios
func (c *HTTPWeatherClient) ExecuteRequest(ctx context.Context, req HTTPWeatherRequest) (*HTTPWeatherResponse, error) {
	resp, err := c.client.Get(req.URL)
	if err != nil {
		return nil, infrastructure.NewExternalAPIError(
			fmt.Sprintf("failed to call %s", req.ProviderName), err)
	}

	if resp.StatusCode == http.StatusNotFound {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn(fmt.Sprintf("Failed to close %s response body", req.ProviderName),
				ports.F(ErrorField, closeErr))
		}
		return nil, ports.NewNotFoundError(CityNotFoundMsg)
	}

	if resp.StatusCode != http.StatusOK {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn(fmt.Sprintf("Failed to close %s response body", req.ProviderName),
				ports.F(ErrorField, closeErr))
		}
		return nil, infrastructure.NewExternalAPIError(
			fmt.Sprintf("%s returned status %d", req.ProviderName, resp.StatusCode), nil)
	}

	return &HTTPWeatherResponse{
		Body:       resp.Body,
		StatusCode: resp.StatusCode,
	}, nil
}

// DecodeJSONResponse decodes JSON response and handles body cleanup
func (c *HTTPWeatherClient) DecodeJSONResponse(resp *HTTPWeatherResponse, target interface{}, providerName string) error {
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn(fmt.Sprintf("Failed to close %s response body", providerName),
				ports.F(ErrorField, closeErr))
		}
	}()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return infrastructure.NewExternalAPIError(
			fmt.Sprintf("failed to decode %s response", providerName), err)
	}

	return nil
}

// ValidateBasicParams validates common parameters for weather providers
func ValidateBasicParams(city, apiKey string) error {
	if city == "" {
		return infrastructure.NewValidationError(EmptyCityValidationMsg)
	}
	if apiKey == "" {
		return infrastructure.NewValidationError(EmptyAPIKeyValidationMsg)
	}
	return nil
}

// ValidateConstructorParams validates parameters during provider construction
func ValidateConstructorParams(apiKey string) error {
	if apiKey == "" {
		return infrastructure.NewValidationError(EmptyAPIKeyValidationMsg)
	}
	return nil
}
