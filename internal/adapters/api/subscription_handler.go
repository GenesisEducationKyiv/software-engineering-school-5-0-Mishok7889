package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/shared"
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
	s.logger.Debug(HandlingSubscriptionMsg)

	if err := c.ShouldBind(&httpReq); err != nil {
		s.logger.Error(RequestBindingErrorMsg, ports.F(ErrorField, err))
		s.handleError(c, NewValidationError(InvalidRequestFormatMsg))
		return
	}

	s.logger.Debug(SubscriptionRequestMsg,
		ports.F(EmailField, httpReq.Email),
		ports.F(CityField, httpReq.City),
		ports.F(FrequencyField, httpReq.Frequency))

	domainReq := subscription.SubscribeParams{
		Email:     httpReq.Email,
		City:      httpReq.City,
		Frequency: subscription.FrequencyFromString(httpReq.Frequency),
	}

	if err := s.subscriptionUseCase.Subscribe(c.Request.Context(), domainReq); err != nil {
		s.logger.Error(SubscriptionErrorMsg,
			ports.F(ErrorField, err),
			ports.F(EmailField, httpReq.Email),
			ports.F(CityField, httpReq.City))
		s.handleError(c, err)
		return
	}

	s.logger.Debug(SubscriptionCreatedMsg,
		ports.F(EmailField, httpReq.Email),
		ports.F(CityField, httpReq.City))
	c.JSON(http.StatusOK, SuccessResponse{Message: SubscriptionSuccessMsg})
}

// confirmSubscription handles GET /api/confirm/:token requests
func (s *HTTPServerAdapter) confirmSubscription(c *gin.Context) {
	token := c.GetString(shared.ValidatedTokenKey)

	s.logger.Debug(ConfirmingSubscriptionMsg, ports.F(TokenField, token))

	confirmParams := subscription.ConfirmParams{
		Token: token,
	}

	if err := s.subscriptionUseCase.ConfirmSubscription(c.Request.Context(), confirmParams); err != nil {
		s.logger.Error(ConfirmationErrorMsg, ports.F(ErrorField, err), ports.F(TokenField, token))
		s.handleError(c, err)
		return
	}

	s.logger.Debug(SubscriptionConfirmedMsg, ports.F(TokenField, token))
	c.JSON(http.StatusOK, SuccessResponse{Message: ConfirmationSuccessMsg})
}

// unsubscribe handles GET /api/unsubscribe/:token requests
func (s *HTTPServerAdapter) unsubscribe(c *gin.Context) {
	token := c.GetString(shared.ValidatedTokenKey)

	s.logger.Debug(UnsubscribingMsg, ports.F(TokenField, token))

	unsubscribeParams := subscription.UnsubscribeParams{
		Token: token,
	}

	if err := s.subscriptionUseCase.Unsubscribe(c.Request.Context(), unsubscribeParams); err != nil {
		s.logger.Error(UnsubscribeErrorMsg, ports.F(ErrorField, err), ports.F(TokenField, token))
		s.handleError(c, err)
		return
	}

	s.logger.Debug(UnsubscribedMsg, ports.F(TokenField, token))
	c.JSON(http.StatusOK, SuccessResponse{Message: UnsubscribeSuccessMsg})
}
