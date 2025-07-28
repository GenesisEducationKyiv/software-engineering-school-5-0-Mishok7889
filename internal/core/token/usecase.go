package token

import (
	"context"
	"fmt"
	"time"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/validation"
)

const (
	// Default values
	defaultSubscriptionID = 1
	defaultUserEmail      = "user@example.com"

	// Log field names
	logFieldToken  = "token"
	logFieldUserID = "userID"

	// Log messages
	logValidatingToken = "Validating token"
	logTokenExpired    = "Token is expired"
	logTokenValidated  = "Token validated successfully"
	logGeneratingToken = "Generating token"
	logTokenGenerated  = "Token generated successfully"
	logRevokingToken   = "Revoking token"
	logTokenRevoked    = "Token revoked successfully"
	logGettingUserInfo = "Getting user info"

	// Error messages
	errTokenRepoRequired      = "token repository is required"
	errTokenGeneratorRequired = "token generator is required"
	errLoggerRequired         = "logger is required"
	errTokenEmpty             = "token cannot be empty"
	errUserIDEmpty            = "user ID cannot be empty"
	errEmailEmpty             = "email cannot be empty"
	errTTLPositive            = "TTL must be positive"
	errTokenNotFound          = "token not found"

	// Error operations
	errFindToken   = "find token: %w"
	errSaveToken   = "save token: %w"
	errDeleteToken = "delete token: %w"
)

type Service interface {
	ValidateToken(ctx context.Context, tokenStr string) (*TokenInfo, error)
	GenerateToken(ctx context.Context, request GenerateTokenRequest) (*GeneratedToken, error)
	RevokeToken(ctx context.Context, tokenStr string) error
	GetUserInfo(ctx context.Context, userID string) (*UserInfo, error)
}

type UseCase struct {
	tokenRepo      ports.TokenRepository
	tokenGenerator ports.TokenGenerator
	logger         ports.Logger
}

type UseCaseDependencies struct {
	TokenRepo      ports.TokenRepository
	TokenGenerator ports.TokenGenerator
	Logger         ports.Logger
}

type TokenInfo struct {
	Valid     bool
	UserID    string
	Email     string
	ExpiresAt time.Time
}

type GenerateTokenRequest struct {
	UserID     string
	Email      string
	TTLSeconds int64
}

type GeneratedToken struct {
	Token     string
	ExpiresAt time.Time
}

type UserInfo struct {
	UserID string
	Email  string
}

var _ Service = (*UseCase)(nil)

func NewUseCase(deps UseCaseDependencies) (*UseCase, error) {
	if deps.TokenRepo == nil {
		return nil, shared.NewValidationError(errTokenRepoRequired)
	}
	if deps.TokenGenerator == nil {
		return nil, shared.NewValidationError(errTokenGeneratorRequired)
	}
	if deps.Logger == nil {
		return nil, shared.NewValidationError(errLoggerRequired)
	}

	return &UseCase{
		tokenRepo:      deps.TokenRepo,
		tokenGenerator: deps.TokenGenerator,
		logger:         deps.Logger,
	}, nil
}

func (uc *UseCase) ValidateToken(ctx context.Context, tokenStr string) (*TokenInfo, error) {
	if !validation.IsNotEmpty(tokenStr) {
		return nil, shared.NewValidationError(errTokenEmpty)
	}

	uc.logger.Debug(logValidatingToken, ports.F(logFieldToken, tokenStr))

	tokenData, err := uc.tokenRepo.FindByToken(ctx, tokenStr)
	if err != nil {
		if ports.IsNotFoundError(err) {
			return &TokenInfo{Valid: false}, nil
		}
		return nil, fmt.Errorf(errFindToken, err)
	}

	if tokenData.IsExpired() {
		uc.logger.Debug(logTokenExpired, ports.F(logFieldToken, tokenStr))
		return &TokenInfo{Valid: false}, nil
	}

	uc.logger.Debug(logTokenValidated, ports.F(logFieldToken, tokenStr))
	return &TokenInfo{
		Valid:     true,
		UserID:    fmt.Sprintf("%d", tokenData.SubscriptionID),
		Email:     defaultUserEmail, // TODO: Get from user service
		ExpiresAt: tokenData.ExpiresAt,
	}, nil
}

func (uc *UseCase) GenerateToken(ctx context.Context, request GenerateTokenRequest) (*GeneratedToken, error) {
	if !validation.IsNotEmpty(request.UserID) {
		return nil, shared.NewValidationError(errUserIDEmpty)
	}
	if !validation.IsNotEmpty(request.Email) {
		return nil, shared.NewValidationError(errEmailEmpty)
	}
	if !validation.IsValidEmail(request.Email) {
		return nil, shared.NewValidationError("invalid email format")
	}
	if request.TTLSeconds <= 0 {
		return nil, shared.NewValidationError(errTTLPositive)
	}

	uc.logger.Debug(logGeneratingToken, ports.F(logFieldUserID, request.UserID))

	token := uc.tokenGenerator.GenerateToken()
	expiresAt := time.Now().Add(time.Duration(request.TTLSeconds) * time.Second)

	tokenData := &ports.TokenData{
		Value:          token,
		SubscriptionID: defaultSubscriptionID, // TODO: Map userID to subscriptionID
		Type:           TypeConfirmation,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
	}

	if err := uc.tokenRepo.Save(ctx, tokenData); err != nil {
		return nil, fmt.Errorf(errSaveToken, err)
	}

	uc.logger.Debug(logTokenGenerated, ports.F(logFieldUserID, request.UserID))
	return &GeneratedToken{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (uc *UseCase) RevokeToken(ctx context.Context, tokenStr string) error {
	if !validation.IsNotEmpty(tokenStr) {
		return shared.NewValidationError(errTokenEmpty)
	}

	uc.logger.Debug(logRevokingToken, ports.F(logFieldToken, tokenStr))

	tokenData, err := uc.tokenRepo.FindByToken(ctx, tokenStr)
	if err != nil {
		if ports.IsNotFoundError(err) {
			return shared.NewNotFoundError(errTokenNotFound)
		}
		return fmt.Errorf(errFindToken, err)
	}

	if err := uc.tokenRepo.Delete(ctx, tokenData); err != nil {
		return fmt.Errorf(errDeleteToken, err)
	}

	uc.logger.Debug(logTokenRevoked, ports.F(logFieldToken, tokenStr))
	return nil
}

func (uc *UseCase) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	if !validation.IsNotEmpty(userID) {
		return nil, shared.NewValidationError(errUserIDEmpty)
	}

	uc.logger.Debug(logGettingUserInfo, ports.F(logFieldUserID, userID))

	// TODO: Implement user lookup logic when user management is added
	return &UserInfo{
		UserID: userID,
		Email:  defaultUserEmail,
	}, nil
}
