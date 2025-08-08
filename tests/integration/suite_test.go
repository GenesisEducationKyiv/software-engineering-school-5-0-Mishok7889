package integration

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// ServiceIntegrationTestSuite is a base suite for service-specific integration tests
// Each service should have its own integration test that focuses on testing the service in isolation
// with mocked external dependencies (other services, databases, external APIs)
type ServiceIntegrationTestSuite struct {
	suite.Suite

	// Test mode - determines how tests run
	testMode string
}

func (s *ServiceIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Set test mode - each service should test in isolation
	s.testMode = "service-isolated"
}

func (s *ServiceIntegrationTestSuite) SetupTest() {
	// No cross-service setup needed - each service tests in isolation
	// Individual service tests should set up their own mocks and dependencies
}

func (s *ServiceIntegrationTestSuite) TearDownTest() {
	// Clean up test data for individual service
	// Individual service tests should clean up their own test data
}

func (s *ServiceIntegrationTestSuite) TearDownSuite() {
	// Final cleanup for individual service
	// Individual service tests should handle their own cleanup
}

func (s *ServiceIntegrationTestSuite) TestBasicFunctionality() {
	// Basic test to ensure the suite setup works
	s.NotEmpty(s.testMode)
	s.Equal("service-isolated", s.testMode)
}

// TestServiceIntegrationSuite runs the base suite (mainly for validation)
// The actual integration tests are in service-specific directories:
// - tests/integration/weather/weather_service_test.go
// - tests/integration/user/user_service_test.go
// - tests/integration/subscription/subscription_service_test.go
// - tests/integration/notification/notification_service_test.go
func TestServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ServiceIntegrationTestSuite))
}
