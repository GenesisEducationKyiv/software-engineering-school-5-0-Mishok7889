package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/ports"
)

const (
	// HTTP response messages
	subscriptionSuccessMsg  = "subscription created successfully"
	confirmationSuccessMsg  = "subscription confirmed successfully"
	unsubscribeSuccessMsg   = "unsubscribed successfully"
	invalidRequestFormatMsg = "invalid request format"

	// Log messages
	handlingSubscriptionMsg   = "handling subscription request"
	subscriptionRequestMsg    = "processing subscription request"
	subscriptionErrorMsg      = "subscription creation failed"
	subscriptionCreatedMsg    = "subscription created successfully"
	confirmingSubscriptionMsg = "confirming subscription"
	confirmationErrorMsg      = "confirmation failed"
	subscriptionConfirmedMsg  = "subscription confirmed successfully"
	unsubscribingMsg          = "processing unsubscribe request"
	unsubscribeErrorMsg       = "unsubscribe failed"
	unsubscribedMsg           = "unsubscribed successfully"
	requestBindingErrorMsg    = "request binding failed"

	// Field names for logging
	emailField     = "email"
	cityField      = "city"
	frequencyField = "frequency"
	tokenField     = "token"
	errorField     = "error"
)

type SubscriptionHandlers struct {
	subscriptionUseCase *subscription.UseCase
	logger              ports.Logger
}

func NewSubscriptionHandlers(subscriptionUC *subscription.UseCase, logger ports.Logger) *SubscriptionHandlers {
	return &SubscriptionHandlers{
		subscriptionUseCase: subscriptionUC,
		logger:              logger,
	}
}

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

// ErrorResponse represents an error HTTP response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Subscribe handles POST /api/subscribe requests
func (h *SubscriptionHandlers) Subscribe(c *gin.Context) {
	var httpReq SubscriptionRequest
	h.logger.Debug(handlingSubscriptionMsg)

	if err := c.ShouldBind(&httpReq); err != nil {
		h.logger.Error(requestBindingErrorMsg, ports.F(errorField, err))
		h.handleError(c, NewValidationError(invalidRequestFormatMsg))
		return
	}

	h.logger.Debug(subscriptionRequestMsg,
		ports.F(emailField, httpReq.Email),
		ports.F(cityField, httpReq.City),
		ports.F(frequencyField, httpReq.Frequency))

	domainReq := subscription.SubscribeParams{
		Email:     httpReq.Email,
		City:      httpReq.City,
		Frequency: subscription.FrequencyFromString(httpReq.Frequency),
	}

	if err := h.subscriptionUseCase.Subscribe(c.Request.Context(), domainReq); err != nil {
		h.logger.Error(subscriptionErrorMsg,
			ports.F(errorField, err),
			ports.F(emailField, httpReq.Email),
			ports.F(cityField, httpReq.City))
		h.handleError(c, err)
		return
	}

	h.logger.Debug(subscriptionCreatedMsg,
		ports.F(emailField, httpReq.Email),
		ports.F(cityField, httpReq.City))
	c.JSON(http.StatusOK, SuccessResponse{Message: subscriptionSuccessMsg})
}

// ConfirmSubscription handles GET /api/confirm/:token requests
func (h *SubscriptionHandlers) ConfirmSubscription(c *gin.Context) {
	token := c.GetString(shared.ValidatedTokenKey)

	h.logger.Debug(confirmingSubscriptionMsg, ports.F(tokenField, token))

	confirmParams := subscription.ConfirmParams{
		Token: token,
	}

	if err := h.subscriptionUseCase.ConfirmSubscription(c.Request.Context(), confirmParams); err != nil {
		h.logger.Error(confirmationErrorMsg, ports.F(errorField, err), ports.F(tokenField, token))
		h.handleError(c, err)
		return
	}

	h.logger.Debug(subscriptionConfirmedMsg, ports.F(tokenField, token))
	c.JSON(http.StatusOK, SuccessResponse{Message: confirmationSuccessMsg})
}

// Unsubscribe handles GET /api/unsubscribe/:token requests
func (h *SubscriptionHandlers) Unsubscribe(c *gin.Context) {
	token := c.GetString(shared.ValidatedTokenKey)

	h.logger.Debug(unsubscribingMsg, ports.F(tokenField, token))

	unsubscribeParams := subscription.UnsubscribeParams{
		Token: token,
	}

	if err := h.subscriptionUseCase.Unsubscribe(c.Request.Context(), unsubscribeParams); err != nil {
		h.logger.Error(unsubscribeErrorMsg, ports.F(errorField, err), ports.F(tokenField, token))
		h.handleError(c, err)
		return
	}

	h.logger.Debug(unsubscribedMsg, ports.F(tokenField, token))
	c.JSON(http.StatusOK, SuccessResponse{Message: unsubscribeSuccessMsg})
}

// ValidationError represents a validation error
type ValidationError struct {
	message string
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{message: message}
}

func (e *ValidationError) Error() string {
	return e.message
}

// handleError handles different types of errors and returns appropriate HTTP responses
func (h *SubscriptionHandlers) handleError(c *gin.Context, err error) {
	if shared.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if shared.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	if shared.IsAlreadyExistsError(err) {
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	// Check for our custom ValidationError type
	if _, ok := err.(*ValidationError); ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Default to internal server error
	h.logger.Error("Unhandled error", ports.F(errorField, err))
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}
