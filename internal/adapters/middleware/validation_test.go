package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// Test the middleware behavior when token parameter is empty
func TestValidationMiddleware_ValidateTokenParam_EmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewValidationMiddleware()
	router := gin.New()

	// Test with a route that accepts optional token (using wildcard)
	router.GET("/test/:token", middleware.ValidateTokenParam(), func(c *gin.Context) {
		token := c.GetString("validated_token")
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	// Test with empty token by using URL-encoded empty string
	req := httptest.NewRequest("GET", "/test/", nil)
	w := httptest.NewRecorder()

	// Since /test/ won't match /test/:token, let's test the actual scenario
	// where someone accesses the endpoint but the token is effectively empty
	// We'll create a context manually to test the middleware logic
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "token", Value: ""}} // Simulate empty token

	// Call middleware directly
	handler := middleware.ValidateTokenParam()
	handler(c)

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

			// Build URL with proper encoding
			testURL := "/test"
			if tt.city != "" {
				v := url.Values{}
				v.Set("city", tt.city)
				testURL += "?" + v.Encode()
			}

			req := httptest.NewRequest("GET", testURL, nil)
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
