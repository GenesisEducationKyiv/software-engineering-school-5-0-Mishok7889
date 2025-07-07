package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"weatherapi.app/pkg/errors"
)

func setupErrorTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	server := &HTTPServerAdapter{}

	router := gin.New()

	// Test endpoints for different error types
	router.GET("/test/validation", func(c *gin.Context) {
		server.handleError(c, errors.NewValidationError("validation failed"))
	})

	router.GET("/test/not-found", func(c *gin.Context) {
		server.handleError(c, errors.NewNotFoundError("resource not found"))
	})

	router.GET("/test/already-exists", func(c *gin.Context) {
		server.handleError(c, errors.NewAlreadyExistsError("resource already exists"))
	})

	router.GET("/test/external-api", func(c *gin.Context) {
		server.handleError(c, errors.NewExternalAPIError("external service failed", nil))
	})

	router.GET("/test/internal", func(c *gin.Context) {
		server.handleError(c, errors.NewDatabaseError("internal server error", nil))
	})

	router.GET("/test/database", func(c *gin.Context) {
		server.handleError(c, errors.NewDatabaseError("database connection failed", nil))
	})

	router.GET("/test/configuration", func(c *gin.Context) {
		server.handleError(c, errors.NewConfigurationError("configuration error", nil))
	})

	router.GET("/test/generic", func(c *gin.Context) {
		server.handleError(c, errors.New(errors.ErrorTypeUnknown, "generic error"))
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

func TestHTTPServerAdapter_HandleError_ExternalAPIError(t *testing.T) {
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
