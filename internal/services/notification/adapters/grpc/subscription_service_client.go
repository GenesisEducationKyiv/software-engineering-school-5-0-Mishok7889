package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

type SubscriptionServiceClient struct {
	conn *grpc.ClientConn
}

type SubscriptionServiceConfig struct {
	Host string
	Port int
}

func NewSubscriptionServiceClient(config SubscriptionServiceConfig) (*SubscriptionServiceClient, error) {
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial subscription service at %s: %w", address, err)
	}

	return &SubscriptionServiceClient{
		conn: conn,
	}, nil
}

func (c *SubscriptionServiceClient) GetConfirmedSubscriptions(ctx context.Context, frequency string) ([]*ports.SubscriptionServiceData, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *SubscriptionServiceClient) GetActiveSubscriptionsByCity(ctx context.Context, city string) ([]shared.Subscription, error) {
	// Mock implementation for now - in real implementation this would call gRPC
	return []shared.Subscription{
		{
			ID:        "1",
			Email:     "test@example.com",
			City:      city,
			Frequency: shared.FrequencyHourly,
			IsActive:  true,
		},
	}, nil
}

func (c *SubscriptionServiceClient) GetActiveSubscriptionsByFrequency(ctx context.Context, frequency shared.Frequency) ([]shared.Subscription, error) {
	// Mock implementation for now - in real implementation this would call gRPC
	return []shared.Subscription{
		{
			ID:        "1",
			Email:     "test@example.com",
			City:      "London",
			Frequency: frequency,
			IsActive:  true,
		},
	}, nil
}

func (c *SubscriptionServiceClient) FindByID(ctx context.Context, id uint) (*ports.SubscriptionServiceData, error) {
	return &ports.SubscriptionServiceData{
		ID:    id,
		Email: "test@example.com",
		City:  "London",
	}, nil
}

func (c *SubscriptionServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
