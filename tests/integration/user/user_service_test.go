package user_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	authpb "weatherapi.app/api/proto/auth"
	"weatherapi.app/internal/core/token"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	usergrpc "weatherapi.app/internal/services/user/adapters/grpc"
)

const (
	testTimeout         = 30 * time.Second
	testPort            = 50052
	serverReadyTimeout  = 10 * time.Second
	serverReadyInterval = 100 * time.Millisecond
	numConcurrentReqs   = 10
	numPerfRequests     = 100
	maxAvgTokenLatency  = 10 * time.Millisecond

	// Test data
	testUserID         = "user123"
	testEmail          = "test@example.com"
	testToken          = "abc123def456"
	testTTLSeconds     = 3600
	testSubscriptionID = 1
)

type UserServiceIntegrationSuite struct {
	suite.Suite

	// Service under test
	grpcHandler *usergrpc.AuthServiceServer
	grpcServer  *grpc.Server
	listener    net.Listener

	// Test client
	clientConn *grpc.ClientConn
	authClient authpb.AuthServiceClient

	// Mockery-generated mocks
	mockTokenRepo      *mocks.TokenRepository
	mockTokenGenerator *mocks.TokenGenerator
	mockLogger         *mocks.Logger
}

func TestUserServiceIntegration(t *testing.T) {
	suite.Run(t, new(UserServiceIntegrationSuite))
}

func (s *UserServiceIntegrationSuite) SetupSuite() {
	s.setupMocks()
	s.createUserApplication()
	s.setupGRPCServer()
	s.setupGRPCClient()
	s.waitForServerReady()
}

func (s *UserServiceIntegrationSuite) TearDownSuite() {
	if s.clientConn != nil {
		_ = s.clientConn.Close()
	}

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *UserServiceIntegrationSuite) setupMocks() {
	// Create mockery-generated mocks
	s.mockTokenRepo = mocks.NewTokenRepository(s.T())
	s.mockTokenGenerator = mocks.NewTokenGenerator(s.T())
	s.mockLogger = mocks.NewLogger(s.T())

	// Set up flexible logger mock expectations (logger is called frequently)
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()
}
func (s *UserServiceIntegrationSuite) createUserApplication() {
	// Create token use case with mocks
	tokenUC, err := token.NewUseCase(token.UseCaseDependencies{
		TokenRepo:      s.mockTokenRepo,
		TokenGenerator: s.mockTokenGenerator,
		Logger:         s.mockLogger,
	})
	s.Require().NoError(err)

	// Create gRPC handler directly (this is what we're testing)
	s.grpcHandler = usergrpc.NewAuthServiceServer(tokenUC)
}

func (s *UserServiceIntegrationSuite) setupGRPCServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort))
	s.Require().NoError(err)
	s.listener = listener

	s.grpcServer = grpc.NewServer()
	authpb.RegisterAuthServiceServer(s.grpcServer, s.grpcHandler)

	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()
}

func (s *UserServiceIntegrationSuite) setupGRPCClient() {
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", testPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	s.clientConn = conn
	s.authClient = authpb.NewAuthServiceClient(conn)
}

func (s *UserServiceIntegrationSuite) waitForServerReady() {
	ctx, cancel := context.WithTimeout(context.Background(), serverReadyTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			s.Fail("Server failed to start within timeout")
			return
		default:
			_, err := s.authClient.GetUserInfo(ctx, &authpb.GetUserInfoRequest{UserId: testUserID})
			if err == nil || status.Code(err) != codes.Unavailable {
				return
			}
			time.Sleep(serverReadyInterval)
		}
	}
}
func (s *UserServiceIntegrationSuite) TestValidateToken() {
	testCases := []struct {
		name           string
		token          string
		expectError    bool
		errorCode      codes.Code
		errorContains  string
		expectedValid  bool
		expectedUserID string
		expectedEmail  string
		setupMocks     func()
	}{
		{
			name:           "ValidToken_NotExpired",
			token:          testToken,
			expectError:    false,
			expectedValid:  true,
			expectedUserID: fmt.Sprintf("%d", testSubscriptionID),
			expectedEmail:  "user@example.com",
			setupMocks: func() {
				tokenData := &ports.TokenData{
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "confirmation",
					ExpiresAt:      time.Now().Add(time.Hour),
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(tokenData, nil).Once()
			},
		},
		{
			name:           "ValidToken_Expired",
			token:          testToken,
			expectError:    false,
			expectedValid:  false,
			expectedUserID: "",
			expectedEmail:  "",
			setupMocks: func() {
				tokenData := &ports.TokenData{
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "confirmation",
					ExpiresAt:      time.Now().Add(-time.Hour), // Expired
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(tokenData, nil).Once()
			},
		},
		{
			name:           "TokenNotFound",
			token:          "nonexistent",
			expectError:    false,
			expectedValid:  false,
			expectedUserID: "",
			expectedEmail:  "",
			setupMocks: func() {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, "nonexistent").Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
		{
			name:          "EmptyToken",
			token:         "",
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "WhitespaceToken",
			token:         "   ",
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "DatabaseError",
			token:         testToken,
			expectError:   true,
			errorCode:     codes.Internal,
			errorContains: "internal server error",
			setupMocks: func() {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(nil, fmt.Errorf("database connection failed")).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &authpb.ValidateTokenRequest{Token: tc.token}
			resp, err := s.authClient.ValidateToken(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				assert.Equal(s.T(), tc.errorCode, status.Code(err))
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)

				assert.Equal(s.T(), tc.expectedValid, resp.Valid)
				if tc.expectedValid {
					assert.Equal(s.T(), tc.expectedUserID, resp.UserId)
					assert.Equal(s.T(), tc.expectedEmail, resp.Email)
					assert.NotNil(s.T(), resp.ExpiresAt)
				}
			}
		})
	}
}
func (s *UserServiceIntegrationSuite) TestGenerateToken() {
	testCases := []struct {
		name          string
		userID        string
		email         string
		ttlSeconds    int64
		expectError   bool
		errorCode     codes.Code
		errorContains string
		setupMocks    func()
	}{
		{
			name:        "ValidRequest_Success",
			userID:      testUserID,
			email:       testEmail,
			ttlSeconds:  testTTLSeconds,
			expectError: false,
			setupMocks: func() {
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
					return tokenData.Value == testToken &&
						tokenData.SubscriptionID == 1 &&
						tokenData.Type == "confirmation" &&
						tokenData.ExpiresAt.After(time.Now())
				})).Return(nil).Once()
			},
		},
		{
			name:          "EmptyUserID",
			userID:        "",
			email:         testEmail,
			ttlSeconds:    testTTLSeconds,
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "user ID is required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "EmptyEmail",
			userID:        testUserID,
			email:         "",
			ttlSeconds:    testTTLSeconds,
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "email is required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "InvalidEmail",
			userID:        testUserID,
			email:         "invalid-email",
			ttlSeconds:    testTTLSeconds,
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "invalid email format",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "InvalidTTL_Zero",
			userID:        testUserID,
			email:         testEmail,
			ttlSeconds:    0,
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "TTL must be positive",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "InvalidTTL_Negative",
			userID:        testUserID,
			email:         testEmail,
			ttlSeconds:    -100,
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "TTL must be positive",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "DatabaseError_SaveFailed",
			userID:        testUserID,
			email:         testEmail,
			ttlSeconds:    testTTLSeconds,
			expectError:   true,
			errorCode:     codes.Internal,
			errorContains: "internal server error",
			setupMocks: func() {
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(fmt.Errorf("database save failed")).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &authpb.GenerateTokenRequest{
				UserId:     tc.userID,
				Email:      tc.email,
				TtlSeconds: tc.ttlSeconds,
			}
			resp, err := s.authClient.GenerateToken(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				assert.Equal(s.T(), tc.errorCode, status.Code(err))
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)

				assert.NotEmpty(s.T(), resp.Token)
				assert.NotNil(s.T(), resp.ExpiresAt)
				assert.True(s.T(), resp.ExpiresAt.AsTime().After(time.Now()))
			}
		})
	}
}
func (s *UserServiceIntegrationSuite) TestRevokeToken() {
	testCases := []struct {
		name            string
		token           string
		expectError     bool
		errorCode       codes.Code
		errorContains   string
		expectedSuccess bool
		setupMocks      func()
	}{
		{
			name:            "ValidToken_Success",
			token:           testToken,
			expectError:     false,
			expectedSuccess: true,
			setupMocks: func() {
				tokenData := &ports.TokenData{
					ID:             1,
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "confirmation",
					ExpiresAt:      time.Now().Add(time.Hour),
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(tokenData, nil).Once()
				s.mockTokenRepo.EXPECT().Delete(mock.Anything, tokenData).Return(nil).Once()
			},
		},
		{
			name:            "TokenNotFound",
			token:           "nonexistent",
			expectError:     false,
			expectedSuccess: false,
			setupMocks: func() {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, "nonexistent").Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
		{
			name:          "EmptyToken",
			token:         "",
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "DatabaseError_FindFailed",
			token:         testToken,
			expectError:   true,
			errorCode:     codes.Internal,
			errorContains: "internal server error",
			setupMocks: func() {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(nil, fmt.Errorf("database connection failed")).Once()
			},
		},
		{
			name:            "DatabaseError_DeleteFailed",
			token:           testToken,
			expectError:     true,
			errorCode:       codes.Internal,
			errorContains:   "internal server error",
			expectedSuccess: false,
			setupMocks: func() {
				tokenData := &ports.TokenData{
					ID:             1,
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "confirmation",
					ExpiresAt:      time.Now().Add(time.Hour),
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(tokenData, nil).Once()
				s.mockTokenRepo.EXPECT().Delete(mock.Anything, tokenData).Return(fmt.Errorf("database delete failed")).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &authpb.RevokeTokenRequest{Token: tc.token}
			resp, err := s.authClient.RevokeToken(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				assert.Equal(s.T(), tc.errorCode, status.Code(err))
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				assert.Equal(s.T(), tc.expectedSuccess, resp.Success)
			}
		})
	}
}
func (s *UserServiceIntegrationSuite) TestGetUserInfo() {
	testCases := []struct {
		name           string
		userID         string
		expectError    bool
		errorCode      codes.Code
		errorContains  string
		expectedUserID string
		expectedEmail  string
		setupMocks     func()
	}{
		{
			name:           "ValidUserID_Success",
			userID:         testUserID,
			expectError:    false,
			expectedUserID: testUserID,
			expectedEmail:  "user@example.com", // Default email from use case
			setupMocks:     func() {},          // No mocks needed for simple user info lookup
		},
		{
			name:          "EmptyUserID",
			userID:        "",
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "user ID is required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "WhitespaceUserID",
			userID:        "   ",
			expectError:   true,
			errorCode:     codes.InvalidArgument,
			errorContains: "user ID is required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &authpb.GetUserInfoRequest{UserId: tc.userID}
			resp, err := s.authClient.GetUserInfo(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				assert.Equal(s.T(), tc.errorCode, status.Code(err))
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)

				assert.Equal(s.T(), tc.expectedUserID, resp.UserId)
				assert.Equal(s.T(), tc.expectedEmail, resp.Email)
			}
		})
	}
}
func (s *UserServiceIntegrationSuite) TestConcurrentTokenOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for concurrent requests
	for i := 0; i < numConcurrentReqs; i++ {
		token := fmt.Sprintf("token_%d", i)

		s.mockTokenGenerator.EXPECT().GenerateToken().Return(token).Once()
		s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
			return tokenData.Value == token &&
				tokenData.SubscriptionID == 1 &&
				tokenData.Type == "confirmation"
		})).Return(nil).Once()
	}

	results := make(chan error, numConcurrentReqs)

	for i := 0; i < numConcurrentReqs; i++ {
		go func(id int) {
			req := &authpb.GenerateTokenRequest{
				UserId:     fmt.Sprintf("user%d", id),
				Email:      fmt.Sprintf("user%d@test.com", id),
				TtlSeconds: testTTLSeconds,
			}
			_, err := s.authClient.GenerateToken(ctx, req)
			results <- err
		}(i)
	}

	for i := 0; i < numConcurrentReqs; i++ {
		select {
		case err := <-results:
			s.Require().NoError(err)
		case <-ctx.Done():
			s.Fail("Concurrent requests timeout")
		}
	}
}

func (s *UserServiceIntegrationSuite) TestTokenValidationPerformance() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for performance test
	validTokenData := &ports.TokenData{
		Value:          testToken,
		SubscriptionID: testSubscriptionID,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}

	// Mock multiple validation calls
	s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(validTokenData, nil).Times(numPerfRequests)

	req := &authpb.ValidateTokenRequest{Token: testToken}

	// Performance test loop
	start := time.Now()

	for i := 0; i < numPerfRequests; i++ {
		resp, err := s.authClient.ValidateToken(ctx, req)
		s.Require().NoError(err)
		s.Require().True(resp.Valid)
	}

	duration := time.Since(start)
	avgDuration := duration / numPerfRequests

	s.T().Logf("Performance: %d token validations in %v (avg: %v per request)",
		numPerfRequests, duration, avgDuration)

	assert.True(s.T(), avgDuration < maxAvgTokenLatency,
		"Average token validation should be under 10ms")
}

func (s *UserServiceIntegrationSuite) TestTokenLifecycleWorkflow() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for complete workflow
	s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
	s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
		return tokenData.Value == testToken
	})).Return(nil).Once()

	// Step 1: Generate token
	generateReq := &authpb.GenerateTokenRequest{
		UserId:     testUserID,
		Email:      testEmail,
		TtlSeconds: testTTLSeconds,
	}
	generateResp, err := s.authClient.GenerateToken(ctx, generateReq)
	s.Require().NoError(err)
	s.Require().NotEmpty(generateResp.Token)

	// Setup mocks for validation
	validTokenData := &ports.TokenData{
		Value:          testToken,
		SubscriptionID: testSubscriptionID,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}
	s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(validTokenData, nil).Once()

	// Step 2: Validate token
	validateReq := &authpb.ValidateTokenRequest{Token: testToken}
	validateResp, err := s.authClient.ValidateToken(ctx, validateReq)
	s.Require().NoError(err)
	s.Require().True(validateResp.Valid)
	s.Require().Equal(fmt.Sprintf("%d", testSubscriptionID), validateResp.UserId)

	// Setup mocks for revocation
	s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(validTokenData, nil).Once()
	s.mockTokenRepo.EXPECT().Delete(mock.Anything, validTokenData).Return(nil).Once()

	// Step 3: Revoke token
	revokeReq := &authpb.RevokeTokenRequest{Token: testToken}
	revokeResp, err := s.authClient.RevokeToken(ctx, revokeReq)
	s.Require().NoError(err)
	s.Require().True(revokeResp.Success)

	// Setup mocks for validation after revocation
	s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(nil, ports.NewNotFoundError("token not found")).Once()

	// Step 4: Validate revoked token
	validateAfterResp, err := s.authClient.ValidateToken(ctx, validateReq)
	s.Require().NoError(err)
	s.Require().False(validateAfterResp.Valid)
}

func (s *UserServiceIntegrationSuite) TestValidationErrorHandling() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	testCases := []struct {
		name        string
		request     interface{}
		method      string
		expectedErr codes.Code
	}{
		{
			name:        "ValidateToken_NilRequest",
			request:     (*authpb.ValidateTokenRequest)(nil),
			method:      "ValidateToken",
			expectedErr: codes.InvalidArgument,
		},
		{
			name:        "GenerateToken_NilRequest",
			request:     (*authpb.GenerateTokenRequest)(nil),
			method:      "GenerateToken",
			expectedErr: codes.InvalidArgument,
		},
		{
			name:        "RevokeToken_NilRequest",
			request:     (*authpb.RevokeTokenRequest)(nil),
			method:      "RevokeToken",
			expectedErr: codes.InvalidArgument,
		},
		{
			name:        "GetUserInfo_NilRequest",
			request:     (*authpb.GetUserInfoRequest)(nil),
			method:      "GetUserInfo",
			expectedErr: codes.InvalidArgument,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var err error
			switch tc.method {
			case "ValidateToken":
				_, err = s.authClient.ValidateToken(ctx, tc.request.(*authpb.ValidateTokenRequest))
			case "GenerateToken":
				_, err = s.authClient.GenerateToken(ctx, tc.request.(*authpb.GenerateTokenRequest))
			case "RevokeToken":
				_, err = s.authClient.RevokeToken(ctx, tc.request.(*authpb.RevokeTokenRequest))
			case "GetUserInfo":
				_, err = s.authClient.GetUserInfo(ctx, tc.request.(*authpb.GetUserInfoRequest))
			}

			s.Require().Error(err)
			assert.Equal(s.T(), tc.expectedErr, status.Code(err))
		})
	}
}
func (s *UserServiceIntegrationSuite) TestEmailValidation() {
	testCases := []struct {
		name          string
		email         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "ValidEmail_Standard",
			email:       "user@example.com",
			expectError: false,
		},
		{
			name:        "ValidEmail_WithDots",
			email:       "user.name@example.com",
			expectError: false,
		},
		{
			name:        "ValidEmail_WithNumbers",
			email:       "user123@example.com",
			expectError: false,
		},
		{
			name:        "ValidEmail_WithSubdomain",
			email:       "user@mail.example.com",
			expectError: false,
		},
		{
			name:          "InvalidEmail_NoAt",
			email:         "userexample.com",
			expectError:   true,
			errorContains: "invalid email format",
		},
		{
			name:          "InvalidEmail_NoDomain",
			email:         "user@",
			expectError:   true,
			errorContains: "invalid email format",
		},
		{
			name:          "InvalidEmail_NoLocal",
			email:         "@example.com",
			expectError:   true,
			errorContains: "invalid email format",
		},
		{
			name:          "InvalidEmail_NoTLD",
			email:         "user@example",
			expectError:   true,
			errorContains: "invalid email format",
		},
		{
			name:          "InvalidEmail_SpecialChars",
			email:         "user@exam$ple.com",
			expectError:   true,
			errorContains: "invalid email format",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Setup mocks only for valid emails
			if !tc.expectError {
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
			}

			req := &authpb.GenerateTokenRequest{
				UserId:     testUserID,
				Email:      tc.email,
				TtlSeconds: testTTLSeconds,
			}
			resp, err := s.authClient.GenerateToken(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				assert.Equal(s.T(), codes.InvalidArgument, status.Code(err))
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				assert.NotEmpty(s.T(), resp.Token)
			}
		})
	}
}

func (s *UserServiceIntegrationSuite) TestTokenEdgeCases() {
	testCases := []struct {
		name          string
		token         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "VeryLongToken",
			token:       strings.Repeat("a", 500),
			expectError: false,
		},
		{
			name:        "TokenWithSpecialChars",
			token:       "token-with_special.chars123",
			expectError: false,
		},
		{
			name:        "TokenWithUnicode",
			token:       "token_αβγ_测试",
			expectError: false,
		},
		{
			name:          "OnlyWhitespace",
			token:         "\t\n\r ",
			expectError:   true,
			errorContains: "required",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Setup mocks only for valid tokens
			if !tc.expectError {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, tc.token).Return(nil, ports.NewNotFoundError("token not found")).Once()
			}

			req := &authpb.ValidateTokenRequest{Token: tc.token}
			resp, err := s.authClient.ValidateToken(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				assert.Equal(s.T(), codes.InvalidArgument, status.Code(err))
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				// Token not found is still a valid response (false validation)
				assert.False(s.T(), resp.Valid)
			}
		})
	}
}
