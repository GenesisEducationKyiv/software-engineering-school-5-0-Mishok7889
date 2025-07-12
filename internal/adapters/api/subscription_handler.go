package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/ports"
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
	s.logger.Debug("Handling subscription request")

	if err := c.ShouldBind(&httpReq); err != nil {
		s.logger.Error("Request binding error", ports.F("error", err))
		s.handleError(c, NewValidationError("Invalid request format"))
		return
	}

	s.logger.Debug("Subscription request received",
		ports.F("email", httpReq.Email),
		ports.F("city", httpReq.City),
		ports.F("frequency", httpReq.Frequency))

	domainReq := subscription.SubscribeParams{
		Email:     httpReq.Email,
		City:      httpReq.City,
		Frequency: subscription.FrequencyFromString(httpReq.Frequency),
	}

	if err := s.subscriptionUseCase.Subscribe(c.Request.Context(), domainReq); err != nil {
		s.logger.Error("Subscription error",
			ports.F("error", err),
			ports.F("email", httpReq.Email),
			ports.F("city", httpReq.City))
		s.handleError(c, err)
		return
	}

	s.logger.Debug("Subscription created successfully",
		ports.F("email", httpReq.Email),
		ports.F("city", httpReq.City))
	c.JSON(http.StatusOK, SuccessResponse{Message: "Subscription successful. Confirmation email sent."})
}

// confirmSubscription handles GET /api/confirm/:token requests
func (s *HTTPServerAdapter) confirmSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		s.handleError(c, NewValidationError("token parameter is required"))
		return
	}

	s.logger.Debug("Confirming subscription", ports.F("token", token))

	confirmParams := subscription.ConfirmParams{
		Token: token,
	}

	if err := s.subscriptionUseCase.ConfirmSubscription(c.Request.Context(), confirmParams); err != nil {
		s.logger.Error("Confirmation error", ports.F("error", err), ports.F("token", token))
		s.handleError(c, err)
		return
	}

	s.logger.Debug("Subscription confirmed successfully", ports.F("token", token))
	c.JSON(http.StatusOK, SuccessResponse{Message: "Subscription confirmed successfully"})
}

// unsubscribe handles GET /api/unsubscribe/:token requests
func (s *HTTPServerAdapter) unsubscribe(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		s.handleError(c, NewValidationError("token parameter is required"))
		return
	}

	s.logger.Debug("Unsubscribing", ports.F("token", token))

	unsubscribeParams := subscription.UnsubscribeParams{
		Token: token,
	}

	if err := s.subscriptionUseCase.Unsubscribe(c.Request.Context(), unsubscribeParams); err != nil {
		s.logger.Error("Unsubscribe error", ports.F("error", err), ports.F("token", token))
		s.handleError(c, err)
		return
	}

	s.logger.Debug("Unsubscribed successfully", ports.F("token", token))
	c.JSON(http.StatusOK, SuccessResponse{Message: "Unsubscribed successfully"})
}
