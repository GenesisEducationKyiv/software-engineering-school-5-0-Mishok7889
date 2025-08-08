package weather_test

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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	weatherpb "weatherapi.app/api/proto/weather"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	weathergrpc "weatherapi.app/internal/services/weather/adapters/grpc"
	"weatherapi.app/pkg/logger"
)

const (
	testTimeout         = 30 * time.Second
	testPort            = 50051
	serverReadyTimeout  = 10 * time.Second
	serverReadyInterval = 100 * time.Millisecond
	numConcurrentReqs   = 10
	maxCachedLatency    = 50 * time.Millisecond

	// Test data
	testLondonTemp      = 15.5
	testLondonHumidity  = 65
	testDefaultTemp     = 20.0
	testDefaultHumidity = 60
)

type WeatherServiceIntegrationSuite struct {
	suite.Suite

	// Service under test
	grpcHandler *weathergrpc.WeatherServiceServer
	grpcServer  *grpc.Server
	listener    net.Listener

	// Test client
	clientConn    *grpc.ClientConn
	weatherClient weatherpb.WeatherServiceClient

	// Mockery-generated mocks
	mockProvider *mocks.WeatherProviderManager
	mockCache    *mocks.WeatherCache
	mockConfig   *mocks.ConfigProvider
	mockLogger   *mocks.Logger
	mockMetrics  *mocks.WeatherMetrics
}

func TestWeatherServiceIntegration(t *testing.T) {
	suite.Run(t, new(WeatherServiceIntegrationSuite))
}

func (s *WeatherServiceIntegrationSuite) SetupSuite() {
	s.setupMocks()
	s.createWeatherApplication()
	s.setupGRPCServer()
	s.setupGRPCClient()
	s.waitForServerReady()
}

func (s *WeatherServiceIntegrationSuite) TearDownSuite() {
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

func (s *WeatherServiceIntegrationSuite) setupMocks() {
	// Create mockery-generated mocks
	s.mockProvider = mocks.NewWeatherProviderManager(s.T())
	s.mockCache = mocks.NewWeatherCache(s.T())
	s.mockConfig = mocks.NewConfigProvider(s.T())
	s.mockLogger = mocks.NewLogger(s.T())
	s.mockMetrics = mocks.NewWeatherMetrics(s.T())

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

	// Set up config mock expectations (called during cache operations)
	s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    5 * time.Minute,
	}).Maybe()

	// Set up metrics mock expectations (called during server readiness check)
	s.mockMetrics.EXPECT().GetProviderInfo().Return(ports.ProviderInfo{
		TotalProviders:  3,
		ProviderOrder:   []string{"weatherapi", "openweathermap", "accuweather"},
		ChainEnabled:    true,
		FallbackEnabled: true,
	}).Maybe()

	// Set up cache metrics mock expectations
	s.mockMetrics.EXPECT().GetCacheMetrics().Return(ports.CacheStats{
		Hits:     10,
		Misses:   5,
		TotalOps: 15,
		HitRatio: 0.67,
	}, nil).Maybe()
}

func (s *WeatherServiceIntegrationSuite) createWeatherApplication() {
	// Create weather use case with mocks
	weatherUC, err := weather.NewUseCase(weather.UseCaseDependencies{
		WeatherProvider: s.mockProvider,
		Cache:           s.mockCache,
		Config:          s.mockConfig,
		Logger:          s.mockLogger,
		Metrics:         s.mockMetrics,
	})
	s.Require().NoError(err)

	// Create a test logger for gRPC handler
	testLogger := logger.NewServiceLogger("weather-service-test", "test", false)

	// Create gRPC handler with both use case and logger (this is what we're testing)
	s.grpcHandler = weathergrpc.NewWeatherServiceServer(weatherUC, testLogger)
}

func (s *WeatherServiceIntegrationSuite) setupGRPCServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort))
	s.Require().NoError(err)
	s.listener = listener

	s.grpcServer = grpc.NewServer()
	weatherpb.RegisterWeatherServiceServer(s.grpcServer, s.grpcHandler)

	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()
}

func (s *WeatherServiceIntegrationSuite) setupGRPCClient() {
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", testPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	s.clientConn = conn
	s.weatherClient = weatherpb.NewWeatherServiceClient(conn)
}

func (s *WeatherServiceIntegrationSuite) waitForServerReady() {
	require.Eventually(s.T(), func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := s.weatherClient.GetProviderInfo(ctx, &weatherpb.GetProviderInfoRequest{})
		return err == nil
	}, serverReadyTimeout, serverReadyInterval, "Server failed to start within timeout")
}

func (s *WeatherServiceIntegrationSuite) TestGetWeather() {
	testCases := []struct {
		name          string
		city          string
		expectError   bool
		errorContains string
		expectedCity  string
		expectedTemp  float64
		expectedHumid float64
		expectedDesc  string
		setupMocks    func()
	}{
		{
			name:          "ValidCity_London",
			city:          "London",
			expectError:   false,
			expectedCity:  "London",
			expectedTemp:  testLondonTemp,
			expectedHumid: float64(testLondonHumidity),
			expectedDesc:  "Partly cloudy",
			setupMocks: func() {
				// Mock config for cache enable
				s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
					EnableCache: true,
					CacheTTL:    5 * time.Minute,
				}).Maybe()

				// Mock cache miss
				s.mockCache.EXPECT().Get(mock.Anything, "weather:London").Return(nil, shared.NewNotFoundError("cache miss")).Once()

				// Mock successful provider response
				s.mockProvider.EXPECT().GetWeather(mock.Anything, "London").Return(&ports.WeatherData{
					Temperature: testLondonTemp,
					Humidity:    float64(testLondonHumidity),
					Description: "Partly cloudy",
					City:        "London",
					Timestamp:   time.Now(),
				}, nil).Once()

				// Mock cache set
				s.mockCache.EXPECT().Set(mock.Anything, "weather:London", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:          "InvalidCity_NotFound",
			city:          "InvalidCity",
			expectError:   true,
			errorContains: "not found",
			setupMocks: func() {
				s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
					EnableCache: true,
					CacheTTL:    5 * time.Minute,
				}).Maybe()

				s.mockCache.EXPECT().Get(mock.Anything, "weather:InvalidCity").Return(nil, shared.NewNotFoundError("cache miss")).Once()
				s.mockProvider.EXPECT().GetWeather(mock.Anything, "InvalidCity").Return(nil, shared.NewNotFoundError("city not found")).Once()
			},
		},
		{
			name:          "EmptyCity_Required",
			city:          "",
			expectError:   true,
			errorContains: "required",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "DefaultCity_Berlin",
			city:          "Berlin",
			expectError:   false,
			expectedCity:  "Berlin",
			expectedTemp:  testDefaultTemp,
			expectedHumid: float64(testDefaultHumidity),
			expectedDesc:  "Clear",
			setupMocks: func() {
				s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
					EnableCache: true,
					CacheTTL:    5 * time.Minute,
				}).Maybe()

				s.mockCache.EXPECT().Get(mock.Anything, "weather:Berlin").Return(nil, shared.NewNotFoundError("cache miss")).Once()
				s.mockProvider.EXPECT().GetWeather(mock.Anything, "Berlin").Return(&ports.WeatherData{
					Temperature: testDefaultTemp,
					Humidity:    float64(testDefaultHumidity),
					Description: "Clear",
					City:        "Berlin",
					Timestamp:   time.Now(),
				}, nil).Once()
				s.mockCache.EXPECT().Set(mock.Anything, "weather:Berlin", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Reset mocks and setup expectations for this test
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &weatherpb.GetWeatherRequest{City: tc.city}
			resp, err := s.weatherClient.GetWeather(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				s.Require().NotNil(resp.Weather)

				weather := resp.Weather
				assert.Equal(s.T(), tc.expectedCity, weather.City)
				assert.Equal(s.T(), tc.expectedTemp, weather.Temperature)
				assert.Equal(s.T(), tc.expectedHumid, weather.Humidity)
				assert.Equal(s.T(), tc.expectedDesc, weather.Description)
				assert.NotNil(s.T(), weather.Timestamp)
			}
		})
	}
}

func (s *WeatherServiceIntegrationSuite) TestGetWeather_InvalidCityNames() {
	testCases := []struct {
		name          string
		city          string
		errorContains string
		setupMocks    func()
	}{
		{
			name:          "SpecialCharacters",
			city:          "City@#$%",
			errorContains: "invalid characters",
			setupMocks:    func() {}, // Validation error, no mocks needed
		},
		{
			name:          "TooLong",
			city:          strings.Repeat("a", 101), // 101 characters
			errorContains: "exceed",
			setupMocks:    func() {}, // Validation error, no mocks needed
		},
		{
			name:          "NumbersOnly_ProviderError",
			city:          "12345",
			errorContains: "not found", // This will fail at provider level
			setupMocks: func() {
				// This passes validation but fails at provider level
				s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
					EnableCache: true,
					CacheTTL:    5 * time.Minute,
				}).Maybe()

				s.mockCache.EXPECT().Get(mock.Anything, "weather:12345").Return(nil, shared.NewNotFoundError("cache miss")).Once()
				s.mockProvider.EXPECT().GetWeather(mock.Anything, "12345").Return(nil, shared.NewNotFoundError("city not found")).Once()
			},
		},
		{
			name:          "OnlySpaces",
			city:          "   ",
			errorContains: "required",
			setupMocks:    func() {}, // Validation error, no mocks needed
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &weatherpb.GetWeatherRequest{City: tc.city}
			resp, err := s.weatherClient.GetWeather(ctx, req)

			s.Require().Error(err)
			s.Require().Nil(resp)
			if tc.errorContains != "" {
				assert.Contains(s.T(), err.Error(), tc.errorContains)
			}
		})
	}
}

func (s *WeatherServiceIntegrationSuite) TestGetWeatherBatch() {
	testCases := []struct {
		name              string
		cities            []string
		expectError       bool
		errorContains     string
		expectedDataCount int
		setupMocks        func()
	}{
		{
			name:              "MultipleCities_Success",
			cities:            []string{"London", "Paris", "Berlin"},
			expectError:       false,
			expectedDataCount: 3,
			setupMocks: func() {
				// Mock config for cache
				s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
					EnableCache: true,
					CacheTTL:    5 * time.Minute,
				}).Maybe()

				cities := []string{"London", "Paris", "Berlin"}
				for _, city := range cities {
					cacheKey := fmt.Sprintf("weather:%s", city)
					weatherData := &ports.WeatherData{
						Temperature: testDefaultTemp,
						Humidity:    float64(testDefaultHumidity),
						Description: "Clear",
						City:        city,
						Timestamp:   time.Now(),
					}

					// Cache miss
					s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, shared.NewNotFoundError("cache miss")).Once()
					// Provider call
					s.mockProvider.EXPECT().GetWeather(mock.Anything, city).Return(weatherData, nil).Once()
					// Cache set
					s.mockCache.EXPECT().Set(mock.Anything, cacheKey, mock.Anything, mock.Anything).Return(nil).Once()
				}
			},
		},
		{
			name:          "EmptyList_Error",
			cities:        []string{},
			expectError:   true,
			errorContains: "at least one city",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:          "TooManyCities_Error",
			cities:        make([]string, 51),
			expectError:   true,
			errorContains: "exceed",
			setupMocks:    func() {}, // No mocks needed for validation errors
		},
		{
			name:              "SingleCity_Success",
			cities:            []string{"London"},
			expectError:       false,
			expectedDataCount: 1,
			setupMocks: func() {
				s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
					EnableCache: true,
					CacheTTL:    5 * time.Minute,
				}).Maybe()

				weatherData := &ports.WeatherData{
					Temperature: testLondonTemp,
					Humidity:    float64(testLondonHumidity),
					Description: "Partly cloudy",
					City:        "London",
					Timestamp:   time.Now(),
				}

				s.mockCache.EXPECT().Get(mock.Anything, "weather:London").Return(nil, shared.NewNotFoundError("cache miss")).Once()
				s.mockProvider.EXPECT().GetWeather(mock.Anything, "London").Return(weatherData, nil).Once()
				s.mockCache.EXPECT().Set(mock.Anything, "weather:London", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
	}

	// Initialize cities for the TooManyCities test case
	for i := range testCases[2].cities {
		testCases[2].cities[i] = fmt.Sprintf("City%d", i)
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			req := &weatherpb.GetWeatherBatchRequest{Cities: tc.cities}
			resp, err := s.weatherClient.GetWeatherBatch(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				s.Require().Nil(resp)
				if tc.errorContains != "" {
					assert.Contains(s.T(), err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				s.Require().Len(resp.WeatherData, tc.expectedDataCount)

				for _, weather := range resp.WeatherData {
					assert.NotEmpty(s.T(), weather.City)
					assert.NotZero(s.T(), weather.Temperature)
					assert.NotZero(s.T(), weather.Humidity)
					assert.NotEmpty(s.T(), weather.Description)
					assert.NotNil(s.T(), weather.Timestamp)
				}
			}
		})
	}
}

func (s *WeatherServiceIntegrationSuite) TestGetProviderInfo() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	req := &weatherpb.GetProviderInfoRequest{}

	resp, err := s.weatherClient.GetProviderInfo(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().NotNil(resp.ProviderInfo)

	providerInfo := resp.ProviderInfo
	assert.True(s.T(), providerInfo.TotalProviders > 0)
	assert.NotEmpty(s.T(), providerInfo.ProviderOrder)
	assert.Contains(s.T(), providerInfo.ProviderOrder, "weatherapi")
}

func (s *WeatherServiceIntegrationSuite) TestGetCacheMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for the GetWeather call that primes the cache
	testCity := "London"
	cacheKey := "weather:London"
	weatherData := &ports.WeatherData{
		Temperature: testLondonTemp,
		Humidity:    float64(testLondonHumidity),
		Description: "Partly cloudy",
		City:        testCity,
		Timestamp:   time.Now(),
	}

	s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    5 * time.Minute,
	}).Maybe()

	// Mock expectations for the GetWeather call
	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, shared.NewNotFoundError("cache miss")).Once()
	s.mockProvider.EXPECT().GetWeather(mock.Anything, testCity).Return(weatherData, nil).Once()
	s.mockCache.EXPECT().Set(mock.Anything, cacheKey, mock.Anything, mock.Anything).Return(nil).Once()

	// Call GetWeather to prime the system
	weatherReq := &weatherpb.GetWeatherRequest{City: testCity}
	_, err := s.weatherClient.GetWeather(ctx, weatherReq)
	s.Require().NoError(err)

	// Now test GetCacheMetrics
	req := &weatherpb.GetCacheMetricsRequest{}
	resp, err := s.weatherClient.GetCacheMetrics(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().NotNil(resp.CacheStats)

	cacheStats := resp.CacheStats
	assert.True(s.T(), cacheStats.TotalOps >= 0)
	assert.True(s.T(), cacheStats.HitRatio >= 0.0 && cacheStats.HitRatio <= 1.0)
}

func (s *WeatherServiceIntegrationSuite) TestCacheBehavior() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for cache behavior test
	testCity := "CacheTestCity"
	expectedWeatherData := &ports.WeatherData{
		Temperature: 18.5,
		Humidity:    70.0,
		Description: "Cache test weather",
		City:        testCity,
		Timestamp:   time.Now(),
	}

	// Mock config for cache enable
	s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    5 * time.Minute,
	}).Maybe()

	// First call expectations: cache miss → provider call → cache set
	s.mockCache.EXPECT().Get(mock.Anything, "weather:CacheTestCity").Return(nil, shared.NewNotFoundError("cache miss")).Once()
	s.mockProvider.EXPECT().GetWeather(mock.Anything, testCity).Return(expectedWeatherData, nil).Once()
	s.mockCache.EXPECT().Set(mock.Anything, "weather:CacheTestCity", mock.Anything, mock.Anything).Return(nil).Once()

	// Second call expectations: cache hit → no provider call
	s.mockCache.EXPECT().Get(mock.Anything, "weather:CacheTestCity").Return(expectedWeatherData, nil).Once()

	req := &weatherpb.GetWeatherRequest{City: testCity}

	// First call - should miss cache and call provider
	resp1, err := s.weatherClient.GetWeather(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp1)

	// Second call - should hit cache and be faster
	start := time.Now()
	resp2, err := s.weatherClient.GetWeather(ctx, req)
	duration := time.Since(start)

	s.Require().NoError(err)
	s.Require().NotNil(resp2)

	assert.Equal(s.T(), resp1.Weather.City, resp2.Weather.City)
	assert.Equal(s.T(), resp1.Weather.Temperature, resp2.Weather.Temperature)
	assert.True(s.T(), duration < maxCachedLatency, "Cached request should be very fast")
}

func (s *WeatherServiceIntegrationSuite) TestConcurrentRequests() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for concurrent requests
	s.mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    5 * time.Minute,
	}).Maybe()

	// Setup expectations for each concurrent request
	for i := 0; i < numConcurrentReqs; i++ {
		city := fmt.Sprintf("City%d", i)
		cacheKey := fmt.Sprintf("weather:%s", city)
		weatherData := &ports.WeatherData{
			Temperature: testDefaultTemp + float64(i), // Vary temperature slightly
			Humidity:    float64(testDefaultHumidity),
			Description: "Clear",
			City:        city,
			Timestamp:   time.Now(),
		}

		s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, shared.NewNotFoundError("cache miss")).Once()
		s.mockProvider.EXPECT().GetWeather(mock.Anything, city).Return(weatherData, nil).Once()
		s.mockCache.EXPECT().Set(mock.Anything, cacheKey, mock.Anything, mock.Anything).Return(nil).Once()
	}

	results := make(chan error, numConcurrentReqs)

	for i := 0; i < numConcurrentReqs; i++ {
		go func(id int) {
			req := &weatherpb.GetWeatherRequest{
				City: fmt.Sprintf("City%d", id),
			}
			_, err := s.weatherClient.GetWeather(ctx, req)
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

func (s *WeatherServiceIntegrationSuite) TestValidationErrors() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	testCases := []struct {
		name          string
		requestType   string
		city          string
		expectError   bool
		errorContains string
	}{
		{
			name:          "GetWeather_EmptyCity",
			requestType:   "GetWeather",
			city:          "",
			expectError:   true,
			errorContains: "required",
		},
		{
			name:          "GetWeather_InvalidCharacters",
			requestType:   "GetWeather",
			city:          "Test@City",
			expectError:   true,
			errorContains: "invalid characters",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			switch tc.requestType {
			case "GetWeather":
				req := &weatherpb.GetWeatherRequest{City: tc.city}
				_, err := s.weatherClient.GetWeather(ctx, req)
				if tc.expectError {
					s.Require().Error(err)
					if tc.errorContains != "" {
						assert.Contains(s.T(), err.Error(), tc.errorContains)
					}
				} else {
					s.Require().NoError(err)
				}
			}
		})
	}
}
