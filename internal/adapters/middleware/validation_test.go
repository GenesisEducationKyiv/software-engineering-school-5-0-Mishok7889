package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestValidationMiddleware_ValidateTokenParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid token",
			url:            "/test/valid-token",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "token with special chars",
			url:            "/test/abc-123-def",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			middleware := NewValidationMiddleware()
			router := gin.New()

			router.GET("/test/:token", middleware.ValidateTokenParam(), func(c *gin.Context) {
				token := c.GetString("validated_token")
				c.JSON(http.StatusOK, gin.H{"token": token})
			})

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			}
		})
	}
}

// Test the middleware behavior when token parameter could be empty
// This tests the middleware logic directly
func TestValidationMiddleware_ValidateTokenParam_EmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewValidationMiddleware()
	router := gin.New()

	// Create a route that can capture empty parameters using wildcard
	router.GET("/test/*path", func(c *gin.Context) {
		// Manually set empty token parameter to test middleware logic
		c.Params = gin.Params{
			{Key: "token", Value: ""},
		}

		// Call the middleware function directly
		handler := middleware.ValidateTokenParam()
		handler(c)

		// This should not be reached if middleware aborts
		c.JSON(http.StatusOK, gin.H{"should": "not reach here"})
	})

	req := httptest.NewRequest("GET", "/test/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, ErrTokenRequired, response.Error)
}

func TestValidationMiddleware_ValidateCityQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		city           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid city",
			city:           "London",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "empty city",
			city:           "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrCityRequired,
		},
		{
			name:           "city with whitespace",
			city:           "  ",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrCityRequired,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			middleware := NewValidationMiddleware()
			router := gin.New()

			router.GET("/test", middleware.ValidateCityQuery(), func(c *gin.Context) {
				city := c.GetString("validated_city")
				c.JSON(http.StatusOK, gin.H{"city": city})
			})

			url := "/test"
			if tt.city != "" {
				url += "?city=" + tt.city
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			}
		})
	}
}
