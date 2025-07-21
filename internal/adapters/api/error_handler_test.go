package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

var ginTestModeOnce sync.Once

func setupErrorTestRouter() *gin.Engine {
	// Ensure gin.SetMode is called only once to avoid race conditions
	ginTestModeOnce.Do(func() {
		gin.SetMode(gin.TestMode)
	})

	server := &HTTPServerAdapter{}

	router := gin.New()

	// Test endpoints for different error types
	router.GET("/test/validation", func(c *gin.Context) {
		server.handleError(c, shared.NewValidationError("validation failed"))
	})

	router.GET("/test/not-found", func(c *gin.Context) {
		server.handleError(c, shared.NewNotFoundError("resource not found"))
	})

	router.GET("/test/already-exists", func(c *gin.Context) {
		server.handleError(c, shared.NewAlreadyExistsError("resource already exists"))
	})

	router.GET("/test/external-api", func(c *gin.Context) {
		server.handleError(c, shared.NewExternalServiceError("external service failed"))
	})

	router.GET("/test/internal", func(c *gin.Context) {
		server.handleError(c, shared.NewDomainError(shared.ErrCodeInternal, "internal server error"))
	})

	router.GET("/test/database", func(c *gin.Context) {
		server.handleError(c, shared.NewDomainError(shared.ErrCodeInternal, "database connection failed"))
	})

	router.GET("/test/configuration", func(c *gin.Context) {
		server.handleError(c, shared.NewDomainError(shared.ErrCodeInternal, "configuration error"))
	})

	router.GET("/test/api-validation", func(c *gin.Context) {
		server.handleError(c, NewValidationError("API validation failed"))
	})

	router.GET("/test/generic", func(c *gin.Context) {
		server.handleError(c, shared.NewDomainError(shared.ErrCodeInternal, "generic error"))
	})

	router.GET("/test/port-not-found", func(c *gin.Context) {
		server.handleError(c, ports.NewNotFoundError("port not found"))
	})

	router.GET("/test/port-already-exists", func(c *gin.Context) {
		server.handleError(c, ports.NewAlreadyExistsError("port already exists"))
	})

	return router
}

func TestHTTPServerAdapter_HandleError(t *testing.T) {
	tests := []struct {
		name           string
		route          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "validation error",
			route:          "/test/validation",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation failed",
		},
		{
			name:           "not found error",
			route:          "/test/not-found",
			expectedStatus: http.StatusNotFound,
			expectedError:  "resource not found",
		},
		{
			name:           "already exists error",
			route:          "/test/already-exists",
			expectedStatus: http.StatusConflict,
			expectedError:  "resource already exists",
		},
		{
			name:           "external service error",
			route:          "/test/external-api",
			expectedStatus: http.StatusServiceUnavailable,
			expectedError:  "External service unavailable",
		},
		{
			name:           "internal error",
			route:          "/test/internal",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "database error",
			route:          "/test/database",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "configuration error",
			route:          "/test/configuration",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "API validation error",
			route:          "/test/api-validation",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "API validation failed",
		},
		{
			name:           "generic error",
			route:          "/test/generic",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "port not found error",
			route:          "/test/port-not-found",
			expectedStatus: http.StatusNotFound,
			expectedError:  "port not found",
		},
		{
			name:           "port already exists error",
			route:          "/test/port-already-exists",
			expectedStatus: http.StatusConflict,
			expectedError:  "port already exists",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := setupErrorTestRouter()

			req := httptest.NewRequest("GET", tt.route, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedError, response.Error)
		})
	}
}

func TestHTTPServerAdapter_HandleError_ResponseStructure(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/validation", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.IsType(t, "", response["error"])
	assert.NotEmpty(t, response["error"])
}
