package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	authpb "weatherapi.app/api/proto/auth"
	"weatherapi.app/internal/ports"
)

const (
	dialTimeout       = 5 * time.Second
	requestTimeout    = 10 * time.Second
	defaultTTLSeconds = 86400 // 24 hours

	errConnectFailed = "connect to user service: %w"
)

type UserServiceClient struct {
	client authpb.AuthServiceClient
	conn   *grpc.ClientConn
}

type UserServiceConfig struct {
	Host string
	Port int
}

func NewUserServiceClient(cfg UserServiceConfig) (*UserServiceClient, error) {
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	_, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf(errConnectFailed, err)
	}

	client := authpb.NewAuthServiceClient(conn)

	return &UserServiceClient{
		client: client,
		conn:   conn,
	}, nil
}
func (c *UserServiceClient) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req := &authpb.ValidateTokenRequest{Token: token}
	resp, err := c.client.ValidateToken(reqCtx, req)
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	return &TokenInfo{
		Valid:     resp.Valid,
		UserID:    resp.UserId,
		Email:     resp.Email,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	}, nil
}

func (c *UserServiceClient) GenerateToken(ctx context.Context, userID, email string) (*GeneratedToken, error) {
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req := &authpb.GenerateTokenRequest{
		UserId:     userID,
		Email:      email,
		TtlSeconds: defaultTTLSeconds,
	}
	resp, err := c.client.GenerateToken(reqCtx, req)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &GeneratedToken{
		Token:     resp.Token,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	}, nil
}

func (c *UserServiceClient) RevokeToken(ctx context.Context, token string) error {
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req := &authpb.RevokeTokenRequest{Token: token}
	_, err := c.client.RevokeToken(reqCtx, req)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}

	return nil
}

func (c *UserServiceClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req := &authpb.GetUserInfoRequest{UserId: userID}
	resp, err := c.client.GetUserInfo(reqCtx, req)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	return &UserInfo{
		UserID: resp.UserId,
		Email:  resp.Email,
	}, nil
}

func (c *UserServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// TokenInfo represents token validation response
type TokenInfo struct {
	Valid     bool
	UserID    string
	Email     string
	ExpiresAt time.Time
}

// GeneratedToken represents token generation response
type GeneratedToken struct {
	Token     string
	ExpiresAt time.Time
}

// UserInfo represents user information
type UserInfo struct {
	UserID string
	Email  string
}

// Implement ports.TokenRepository interface for compatibility
func (c *UserServiceClient) FindByToken(ctx context.Context, token string) (*ports.TokenData, error) {
	tokenInfo, err := c.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if !tokenInfo.Valid {
		return nil, ports.NewNotFoundError("token not found")
	}

	return &ports.TokenData{
		Value:          token,
		SubscriptionID: 1, // Default subscription ID mapping
		Type:           "confirmation",
		ExpiresAt:      tokenInfo.ExpiresAt,
		CreatedAt:      time.Now(),
	}, nil
}

func (c *UserServiceClient) Save(ctx context.Context, tokenData *ports.TokenData) error {
	// Map subscription ID to user ID for token generation
	userID := fmt.Sprintf("%d", tokenData.SubscriptionID)
	email := "user@example.com" // Default email, could be passed in tokenData

	generatedToken, err := c.GenerateToken(ctx, userID, email)
	if err != nil {
		return err
	}

	// Update the tokenData with generated values
	tokenData.Value = generatedToken.Token
	tokenData.ExpiresAt = generatedToken.ExpiresAt
	return nil
}

func (c *UserServiceClient) Delete(ctx context.Context, tokenData *ports.TokenData) error {
	return c.RevokeToken(ctx, tokenData.Value)
}
