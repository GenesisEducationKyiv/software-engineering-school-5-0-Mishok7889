package grpc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	weatherpb "weatherapi.app/api/proto/weather"
	"weatherapi.app/internal/core/shared"
	weathercore "weatherapi.app/internal/core/weather"
)

const (
	// Request validation constants
	maxCityNameLength   = 100
	maxBatchRequestSize = 50
	minCityNameLength   = 1

	// City name validation pattern: letters, numbers, spaces, hyphens, dots, apostrophes, and unicode
	cityNamePattern = `^[a-zA-Z0-9\s\-\.'\p{L}]+$`

	// Error messages
	errUseCaseRequired = "weather use case is required"
	errRequestRequired = "request cannot be nil"
	errCityRequired    = "city name is required"
	errCityTooLong     = "city name cannot exceed %d characters"
	errCityTooShort    = "city name must be at least %d character"
	errBatchTooLarge   = "batch request cannot exceed %d cities"
	errBatchEmpty      = "batch request must contain at least one city"
	errInvalidCity     = "city name contains invalid characters"
)

var (
	cityNameRegex = regexp.MustCompile(cityNamePattern)
)

type WeatherServiceServer struct {
	weatherpb.UnimplementedWeatherServiceServer
	weatherUseCase weathercore.Service
}

func NewWeatherServiceServer(weatherUC weathercore.Service) *WeatherServiceServer {
	if weatherUC == nil {
		panic(errUseCaseRequired)
	}
	return &WeatherServiceServer{
		weatherUseCase: weatherUC,
	}
}

func (s *WeatherServiceServer) GetWeather(ctx context.Context, req *weatherpb.GetWeatherRequest) (*weatherpb.GetWeatherResponse, error) {
	if err := validateGetWeatherRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	weatherReq := weathercore.WeatherRequest{
		City: strings.TrimSpace(req.City),
	}

	weatherData, err := s.weatherUseCase.GetWeather(ctx, weatherReq)
	if err != nil {
		return nil, convertDomainError(err)
	}

	return &weatherpb.GetWeatherResponse{
		Weather: convertToProtobufWeatherData(weatherData),
	}, nil
}

func (s *WeatherServiceServer) GetWeatherBatch(ctx context.Context, req *weatherpb.GetWeatherBatchRequest) (*weatherpb.GetWeatherBatchResponse, error) {
	if err := validateGetWeatherBatchRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var weatherDataList []*weatherpb.WeatherData

	for _, city := range req.Cities {
		cityName := strings.TrimSpace(city)
		if cityName == "" {
			continue
		}

		weatherReq := weathercore.WeatherRequest{
			City: cityName,
		}

		weatherData, err := s.weatherUseCase.GetWeather(ctx, weatherReq)
		if err != nil {
			continue
		}

		weatherDataList = append(weatherDataList, convertToProtobufWeatherData(weatherData))
	}

	return &weatherpb.GetWeatherBatchResponse{
		WeatherData: weatherDataList,
	}, nil
}

func (s *WeatherServiceServer) GetProviderInfo(ctx context.Context, req *weatherpb.GetProviderInfoRequest) (*weatherpb.GetProviderInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, errRequestRequired)
	}

	providerInfo := s.weatherUseCase.GetProviderInfo(ctx)

	return &weatherpb.GetProviderInfoResponse{
		ProviderInfo: &weatherpb.ProviderInfo{
			TotalProviders:  int32(providerInfo.TotalProviders),
			ProviderOrder:   providerInfo.ProviderOrder,
			ChainEnabled:    providerInfo.ChainEnabled,
			FallbackEnabled: providerInfo.FallbackEnabled,
		},
	}, nil
}

func (s *WeatherServiceServer) GetCacheMetrics(ctx context.Context, req *weatherpb.GetCacheMetricsRequest) (*weatherpb.GetCacheMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, errRequestRequired)
	}

	cacheStats, err := s.weatherUseCase.GetCacheMetrics(ctx)
	if err != nil {
		return nil, convertDomainError(err)
	}

	return &weatherpb.GetCacheMetricsResponse{
		CacheStats: &weatherpb.CacheStats{
			Hits:     cacheStats.Hits,
			Misses:   cacheStats.Misses,
			TotalOps: cacheStats.TotalOps,
			HitRatio: cacheStats.HitRatio,
		},
	}, nil
}

func validateGetWeatherRequest(req *weatherpb.GetWeatherRequest) error {
	if req == nil {
		return errors.New(errRequestRequired)
	}
	return validateCityName(req.City)
}

func validateGetWeatherBatchRequest(req *weatherpb.GetWeatherBatchRequest) error {
	if req == nil {
		return errors.New(errRequestRequired)
	}
	if len(req.Cities) == 0 {
		return errors.New(errBatchEmpty)
	}
	if len(req.Cities) > maxBatchRequestSize {
		return fmt.Errorf(errBatchTooLarge, maxBatchRequestSize)
	}

	for _, city := range req.Cities {
		if err := validateCityName(city); err != nil {
			return err
		}
	}
	return nil
}

func validateCityName(city string) error {
	city = strings.TrimSpace(city)
	if city == "" {
		return errors.New(errCityRequired)
	}
	if len(city) < minCityNameLength {
		return fmt.Errorf(errCityTooShort, minCityNameLength)
	}
	if len(city) > maxCityNameLength {
		return fmt.Errorf(errCityTooLong, maxCityNameLength)
	}
	if !cityNameRegex.MatchString(city) {
		return errors.New(errInvalidCity)
	}
	return nil
}

func convertToProtobufWeatherData(weatherData *weathercore.Weather) *weatherpb.WeatherData {
	return &weatherpb.WeatherData{
		Temperature: weatherData.Temperature,
		Humidity:    weatherData.Humidity,
		Description: weatherData.Description,
		City:        weatherData.City,
		Timestamp:   timestamppb.New(weatherData.Timestamp),
	}
}

func convertDomainError(err error) error {
	if shared.IsValidationError(err) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if shared.IsNotFoundError(err) {
		return status.Error(codes.NotFound, err.Error())
	}
	return status.Error(codes.Internal, "internal server error")
}
