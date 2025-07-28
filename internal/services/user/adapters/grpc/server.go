package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	authpb "weatherapi.app/api/proto/auth"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/core/token"
	"weatherapi.app/pkg/validation"
)

// TODO: Import when protobuf is generated
// authpb "weatherapi.app/api/proto/auth"

const (
	errUseCaseRequired = "token use case is required"
	errRequestRequired = "request cannot be nil"
	errTokenRequired   = "token is required"
	errUserIDRequired  = "user ID is required"
	errEmailRequired   = "email is required"
	errEmailInvalid    = "invalid email format"
	errTTLRequired     = "TTL must be positive"
	errInternalServer  = "internal server error"
)

// AuthServiceServer implements the AuthService gRPC interface
type AuthServiceServer struct {
	authpb.UnimplementedAuthServiceServer
	tokenUseCase token.Service
}

func NewAuthServiceServer(tokenUC token.Service) *AuthServiceServer {
	if tokenUC == nil {
		panic(errUseCaseRequired)
	}
	return &AuthServiceServer{
		tokenUseCase: tokenUC,
	}
}

func (s *AuthServiceServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, errRequestRequired)
	}

	tokenStr := req.GetToken()
	if !validation.IsNotEmpty(tokenStr) {
		return nil, status.Error(codes.InvalidArgument, errTokenRequired)
	}

	tokenInfo, err := s.tokenUseCase.ValidateToken(ctx, tokenStr)
	if err != nil {
		return nil, convertDomainError(err)
	}

	response := &authpb.ValidateTokenResponse{
		Valid:  tokenInfo.Valid,
		UserId: tokenInfo.UserID,
		Email:  tokenInfo.Email,
	}

	if tokenInfo.Valid {
		response.ExpiresAt = timestamppb.New(tokenInfo.ExpiresAt)
	}

	return response, nil
}

func (s *AuthServiceServer) GenerateToken(ctx context.Context, req *authpb.GenerateTokenRequest) (*authpb.GenerateTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, errRequestRequired)
	}

	userID := req.GetUserId()
	if !validation.IsNotEmpty(userID) {
		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)
	}

	email := req.GetEmail()
	if !validation.IsNotEmpty(email) {
		return nil, status.Error(codes.InvalidArgument, errEmailRequired)
	}
	if !validation.IsValidEmail(email) {
		return nil, status.Error(codes.InvalidArgument, errEmailInvalid)
	}

	if req.GetTtlSeconds() <= 0 {
		return nil, status.Error(codes.InvalidArgument, errTTLRequired)
	}

	generateReq := token.GenerateTokenRequest{
		UserID:     userID,
		Email:      email,
		TTLSeconds: req.GetTtlSeconds(),
	}

	generatedToken, err := s.tokenUseCase.GenerateToken(ctx, generateReq)
	if err != nil {
		return nil, convertDomainError(err)
	}

	return &authpb.GenerateTokenResponse{
		Token:     generatedToken.Token,
		ExpiresAt: timestamppb.New(generatedToken.ExpiresAt),
	}, nil
}

func (s *AuthServiceServer) RevokeToken(ctx context.Context, req *authpb.RevokeTokenRequest) (*authpb.RevokeTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, errRequestRequired)
	}

	tokenStr := req.GetToken()
	if !validation.IsNotEmpty(tokenStr) {
		return nil, status.Error(codes.InvalidArgument, errTokenRequired)
	}

	err := s.tokenUseCase.RevokeToken(ctx, tokenStr)
	if err != nil {
		if shared.IsNotFoundError(err) {
			return &authpb.RevokeTokenResponse{Success: false}, nil
		}
		return nil, convertDomainError(err)
	}

	return &authpb.RevokeTokenResponse{Success: true}, nil
}

func (s *AuthServiceServer) GetUserInfo(ctx context.Context, req *authpb.GetUserInfoRequest) (*authpb.GetUserInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, errRequestRequired)
	}

	userID := req.GetUserId()
	if !validation.IsNotEmpty(userID) {
		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)
	}

	userInfo, err := s.tokenUseCase.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, convertDomainError(err)
	}

	return &authpb.GetUserInfoResponse{
		UserId: userInfo.UserID,
		Email:  userInfo.Email,
	}, nil
}

func convertDomainError(err error) error {
	if shared.IsValidationError(err) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if shared.IsNotFoundError(err) {
		return status.Error(codes.NotFound, err.Error())
	}
	return status.Error(codes.Internal, errInternalServer)
}
