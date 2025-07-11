package api

import (
	"net/http"

	"log/slog"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/pkg/errors"
)

// SubscriptionRequest represents the HTTP request for creating a subscription
type SubscriptionRequest struct {
	Email     string `json:"email" form:"email" binding:"required,email"`
	City      string `json:"city" form:"city" binding:"required"`
	Frequency string `json:"frequency" form:"frequency" binding:"required,oneof=hourly daily"`
}

// SuccessResponse represents a successful HTTP response
type SuccessResponse struct {
	Message string `json:"message"`
}

// subscribe handles POST /api/subscribe requests
func (s *HTTPServerAdapter) subscribe(c *gin.Context) {
	var httpReq SubscriptionRequest
	slog.Debug("Handling subscription request")

	if err := c.ShouldBind(&httpReq); err != nil {
		slog.Error("Request binding error", "error", err)
		// Always use our custom error message to ensure consistency
		s.handleError(c, errors.NewValidationError("Invalid request format"))
		return
	}

	slog.Debug("Subscription request received", "email", httpReq.Email, "city", httpReq.City, "frequency", httpReq.Frequency)

	domainReq := subscription.SubscribeParams{
		Email:     httpReq.Email,
		City:      httpReq.City,
		Frequency: subscription.FrequencyFromString(httpReq.Frequency),
	}

	if err := s.subscriptionUseCase.Subscribe(c.Request.Context(), domainReq); err != nil {
		slog.Error("Subscription error", "error", err, "email", httpReq.Email, "city", httpReq.City)
		s.handleError(c, err)
		return
	}

	slog.Debug("Subscription created successfully", "email", httpReq.Email, "city", httpReq.City)
	c.JSON(http.StatusOK, SuccessResponse{Message: "Subscription successful. Confirmation email sent."})
}

// confirmSubscription handles GET /api/confirm/:token requests
func (s *HTTPServerAdapter) confirmSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		s.handleError(c, errors.NewValidationError("token parameter is required"))
		return
	}

	slog.Debug("Confirming subscription", "token", token)

	confirmParams := subscription.ConfirmParams{
		Token: token,
	}

	if err := s.subscriptionUseCase.ConfirmSubscription(c.Request.Context(), confirmParams); err != nil {
		slog.Error("Confirmation error", "error", err, "token", token)
		s.handleError(c, err)
		return
	}

	slog.Debug("Subscription confirmed successfully", "token", token)
	c.JSON(http.StatusOK, SuccessResponse{Message: "Subscription confirmed successfully"})
}

// unsubscribe handles GET /api/unsubscribe/:token requests
func (s *HTTPServerAdapter) unsubscribe(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		s.handleError(c, errors.NewValidationError("token parameter is required"))
		return
	}

	slog.Debug("Unsubscribing", "token", token)

	unsubscribeParams := subscription.UnsubscribeParams{
		Token: token,
	}

	if err := s.subscriptionUseCase.Unsubscribe(c.Request.Context(), unsubscribeParams); err != nil {
		slog.Error("Unsubscribe error", "error", err, "token", token)
		s.handleError(c, err)
		return
	}

	slog.Debug("Unsubscribed successfully", "token", token)
	c.JSON(http.StatusOK, SuccessResponse{Message: "Unsubscribed successfully"})
}
