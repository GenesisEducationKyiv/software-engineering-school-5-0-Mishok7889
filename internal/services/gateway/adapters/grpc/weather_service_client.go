package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	weatherpb "weatherapi.app/api/proto/weather"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

type WeatherServiceClient struct {
	client weatherpb.WeatherServiceClient
	conn   *grpc.ClientConn
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

	client := weatherpb.NewWeatherServiceClient(conn)

	return &WeatherServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *WeatherServiceClient) GetWeather(ctx context.Context, city string) (*ports.WeatherServiceData, error) {
	req := &weatherpb.GetWeatherRequest{
		City: city,
	}

	resp, err := c.client.GetWeather(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get weather for city %s: %w", city, err)
	}

	return &ports.WeatherServiceData{
		City:        resp.Weather.City,
		Temperature: resp.Weather.Temperature,
		Humidity:    resp.Weather.Humidity,
		Description: resp.Weather.Description,
		Timestamp:   resp.Weather.Timestamp.AsTime(),
	}, nil
}

func (c *WeatherServiceClient) GetWeatherByCity(ctx context.Context, city string) (*shared.WeatherData, error) {
	req := &weatherpb.GetWeatherRequest{
		City: city,
	}

	resp, err := c.client.GetWeather(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get weather for city %s: %w", city, err)
	}

	return &shared.WeatherData{
		City:        resp.Weather.City,
		Temperature: resp.Weather.Temperature,
		Humidity:    resp.Weather.Humidity,
		Description: resp.Weather.Description,
		LastUpdated: resp.Weather.Timestamp.AsTime(),
	}, nil
}

func (c *WeatherServiceClient) GetProviderInfo(ctx context.Context) ports.ProviderInfo {
	return ports.ProviderInfo{
		TotalProviders:  3,
		ProviderOrder:   []string{"weatherapi", "openweathermap", "accuweather"},
		ChainEnabled:    true,
		FallbackEnabled: true,
	}
}

func (c *WeatherServiceClient) Close() error {
	return c.conn.Close()
}
