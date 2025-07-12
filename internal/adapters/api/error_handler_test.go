package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

func setupErrorTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

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

func TestHTTPServerAdapter_HandleError_ValidationError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/validation", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation failed", response.Error)
}

func TestHTTPServerAdapter_HandleError_NotFoundError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/not-found", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "resource not found", response.Error)
}

func TestHTTPServerAdapter_HandleError_AlreadyExistsError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/already-exists", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "resource already exists", response.Error)
}

func TestHTTPServerAdapter_HandleError_ExternalServiceError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/external-api", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "External service unavailable", response.Error)
}

func TestHTTPServerAdapter_HandleError_APIValidationError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/api-validation", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "API validation failed", response.Error)
}

func TestHTTPServerAdapter_HandleError_InternalError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/internal", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Internal server error", response.Error)
}

func TestHTTPServerAdapter_HandleError_DatabaseError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/database", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Internal server error", response.Error)
}

func TestHTTPServerAdapter_HandleError_ConfigurationError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/configuration", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Internal server error", response.Error)
}

func TestHTTPServerAdapter_HandleError_GenericError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/generic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Internal server error", response.Error)
}

func TestHTTPServerAdapter_HandleError_PortNotFoundError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/port-not-found", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "port not found", response.Error)
}

func TestHTTPServerAdapter_HandleError_PortAlreadyExistsError(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/port-already-exists", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "port already exists", response.Error)
}

func TestHTTPServerAdapter_HandleError_ErrorResponseStructure(t *testing.T) {
	router := setupErrorTestRouter()

	req := httptest.NewRequest("GET", "/test/validation", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify the response structure
	assert.Contains(t, response, "error")
	assert.IsType(t, "", response["error"])
	assert.NotEmpty(t, response["error"])
}
