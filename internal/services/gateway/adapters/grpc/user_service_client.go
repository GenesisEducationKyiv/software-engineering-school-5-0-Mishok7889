package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	authpb "weatherapi.app/api/proto/auth"
)

type UserServiceClient struct {
	client authpb.AuthServiceClient
	conn   *grpc.ClientConn
}

type UserServiceConfig struct {
	Host string
	Port int
}

func NewUserServiceClient(config UserServiceConfig) (*UserServiceClient, error) {
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial user service at %s: %w", address, err)
	}

	client := authpb.NewAuthServiceClient(conn)

	return &UserServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *UserServiceClient) ValidateToken(ctx context.Context, token string) (bool, error) {
	req := &authpb.ValidateTokenRequest{
		Token: token,
	}

	resp, err := c.client.ValidateToken(ctx, req)
	if err != nil {
		return false, fmt.Errorf("validate token: %w", err)
	}

	return resp.Valid, nil
}

func (c *UserServiceClient) GenerateToken(ctx context.Context, userID, email string, ttlSeconds int64) (string, error) {
	req := &authpb.GenerateTokenRequest{
		UserId:     userID,
		Email:      email,
		TtlSeconds: ttlSeconds,
	}

	resp, err := c.client.GenerateToken(ctx, req)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return resp.Token, nil
}

func (c *UserServiceClient) GetUserInfo(ctx context.Context, userID string) (*authpb.GetUserInfoResponse, error) {
	req := &authpb.GetUserInfoRequest{
		UserId: userID,
	}

	resp, err := c.client.GetUserInfo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get user info for %s: %w", userID, err)
	}

	return resp, nil
}

func (c *UserServiceClient) Close() error {
	return c.conn.Close()
}
