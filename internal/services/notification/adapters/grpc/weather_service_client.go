package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

type WeatherServiceClient struct {
	conn *grpc.ClientConn
}

type WeatherServiceConfig struct {
	Host string
	Port int
}

func NewWeatherServiceClient(config WeatherServiceConfig) (*WeatherServiceClient, error) {
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial weather service at %s: %w", address, err)
	}

	return &WeatherServiceClient{
		conn: conn,
	}, nil
}

func (c *WeatherServiceClient) GetWeather(ctx context.Context, city string) (*ports.WeatherServiceData, error) {
	// Mock implementation for now - in real implementation this would call gRPC
	return &ports.WeatherServiceData{
		City:        city,
		Temperature: 20.0,
		Humidity:    60.0,
		Description: "Clear sky",
		Timestamp:   time.Now(),
	}, nil
}

func (c *WeatherServiceClient) GetWeatherByCity(ctx context.Context, city string) (*shared.WeatherData, error) {
	// Mock implementation for now - in real implementation this would call gRPC
	return &shared.WeatherData{
		City:        city,
		Temperature: 20.0,
		Humidity:    60.0,
		Description: "Clear sky",
		LastUpdated: time.Now(),
	}, nil
}

func (c *WeatherServiceClient) GetProviderInfo(ctx context.Context) ports.ProviderInfo {
	return ports.ProviderInfo{
		TotalProviders:  1,
		ProviderOrder:   []string{"weather-service-client"},
		ChainEnabled:    false,
		FallbackEnabled: false,
	}
}

func (c *WeatherServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
