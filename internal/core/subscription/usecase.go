package subscription

import (
	"context"
	"fmt"
	"time"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
	"weatherapi.app/pkg/validation"
)

type UseCase struct {
	subscriptionRepo ports.SubscriptionRepository
	tokenRepo        ports.TokenRepository
	emailProvider    ports.EmailProvider
	config           ports.ConfigProvider
	logger           ports.Logger
}

type UseCaseDependencies struct {
	SubscriptionRepo ports.SubscriptionRepository
	TokenRepo        ports.TokenRepository
	EmailProvider    ports.EmailProvider
	Config           ports.ConfigProvider
	Logger           ports.Logger
}

type SubscribeParams struct {
	Email     string
	City      string
	Frequency Frequency
}

type ConfirmParams struct {
	Token string
}

type UnsubscribeParams struct {
	Token string
}

func NewUseCase(deps UseCaseDependencies) (*UseCase, error) {
	if deps.SubscriptionRepo == nil {
		return nil, errors.NewValidationError("subscription repository is required")
	}
	if deps.TokenRepo == nil {
		return nil, errors.NewValidationError("token repository is required")
	}
	if deps.EmailProvider == nil {
		return nil, errors.NewValidationError("email provider is required")
	}
	if deps.Config == nil {
		return nil, errors.NewValidationError("config is required")
	}
	if deps.Logger == nil {
		return nil, errors.NewValidationError("logger is required")
	}

	return &UseCase{
		subscriptionRepo: deps.SubscriptionRepo,
		tokenRepo:        deps.TokenRepo,
		emailProvider:    deps.EmailProvider,
		config:           deps.Config,
		logger:           deps.Logger,
	}, nil
}

func (uc *UseCase) validateSubscribeParams(params SubscribeParams) error {
	// Validate email format
	if !validation.IsNotEmpty(params.Email) {
		return errors.NewValidationError("email is required")
	}
	if !validation.IsValidEmail(params.Email) {
		return errors.NewValidationError("invalid email format")
	}

	// Validate city
	if !validation.IsNotEmpty(params.City) {
		return errors.NewValidationError("city is required")
	}

	// Validate frequency
	if !params.Frequency.IsValid() {
		return errors.NewValidationError("invalid frequency")
	}

	return nil
}

func (uc *UseCase) Subscribe(ctx context.Context, params SubscribeParams) error {
	// Validate input parameters first (fail-fast)
	if err := uc.validateSubscribeParams(params); err != nil {
		return err
	}

	uc.logger.Debug("Processing subscription",
		ports.F("email", params.Email),
		ports.F("city", params.City),
		ports.F("frequency", params.Frequency))

	existing, err := uc.subscriptionRepo.FindByEmail(ctx, params.Email, params.City)
	if err != nil && !errors.IsNotFoundError(err) {
		return fmt.Errorf("check existing subscription: %w", err)
	}

	if existing != nil {
		subscription := uc.convertFromPortsSubscription(existing)
		if subscription.IsConfirmed() {
			return errors.NewAlreadyExistsError("already subscribed")
		}

		// If subscription exists but is not confirmed, update it with new parameters
		if !subscription.IsExpired() {
			uc.logger.Debug("Updating existing unconfirmed subscription",
				ports.F("subscriptionID", existing.ID),
				ports.F("oldFrequency", existing.Frequency),
				ports.F("newFrequency", params.Frequency))

			// Update the existing subscription with new frequency
			existing.Frequency = params.Frequency.String()
			existing.UpdatedAt = time.Now()

			if err := uc.subscriptionRepo.Update(ctx, existing); err != nil {
				return fmt.Errorf("update existing subscription: %w", err)
			}

			// Create updated subscription entity for email
			updatedSubscription := uc.convertFromPortsSubscription(existing)

			// Send new confirmation email
			if err := uc.sendConfirmationEmail(ctx, updatedSubscription); err != nil {
				uc.logger.Error("Failed to send confirmation email for updated subscription",
					ports.F("error", err),
					ports.F("email", params.Email))
				return fmt.Errorf("send confirmation email: %w", err)
			}

			uc.logger.Debug("Existing subscription updated successfully",
				ports.F("email", params.Email),
				ports.F("city", params.City),
				ports.F("frequency", params.Frequency))
			return nil
		}

		// If expired, delete the old subscription
		if err := uc.subscriptionRepo.Delete(ctx, existing); err != nil {
			uc.logger.Warn("Failed to delete expired subscription", ports.F("error", err))
		}
	}

	subscription := NewSubscription(params.Email, params.City, params.Frequency)
	subscriptionData := uc.convertToPortsSubscription(subscription)
	if err := uc.subscriptionRepo.Save(ctx, subscriptionData); err != nil {
		return fmt.Errorf("save subscription: %w", err)
	}

	// Update the subscription entity with the ID from the database
	subscription.ID = subscriptionData.ID
	uc.logger.Debug("Updated subscription with database ID",
		ports.F("subscriptionID", subscription.ID))

	if err := uc.sendConfirmationEmail(ctx, subscription); err != nil {
		uc.logger.Error("Failed to send confirmation email",
			ports.F("error", err),
			ports.F("email", params.Email))
		return fmt.Errorf("send confirmation email: %w", err)
	}

	uc.logger.Debug("Subscription created successfully",
		ports.F("email", params.Email),
		ports.F("city", params.City))
	return nil
}

func (uc *UseCase) ConfirmSubscription(ctx context.Context, params ConfirmParams) error {
	if params.Token == "" {
		return errors.NewValidationError("token is required")
	}

	uc.logger.Debug("Confirming subscription", ports.F("token", params.Token))

	tokenData, err := uc.tokenRepo.FindByToken(ctx, params.Token)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return errors.NewTokenError("invalid or expired confirmation token")
		}
		return fmt.Errorf("find token: %w", err)
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return errors.NewTokenError("invalid or expired confirmation token")
	}

	if tokenData.Type != "confirmation" {
		return errors.NewTokenError("invalid token type")
	}

	subscriptionData, err := uc.subscriptionRepo.FindByID(ctx, tokenData.SubscriptionID)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return errors.NewNotFoundError("subscription not found")
		}
		return fmt.Errorf("find subscription: %w", err)
	}

	subscription := uc.convertFromPortsSubscription(subscriptionData)
	if subscription.IsConfirmed() {
		return errors.NewAlreadyExistsError("subscription is already confirmed")
	}

	subscription.Confirm()
	updatedData := uc.convertToPortsSubscription(subscription)
	if err := uc.subscriptionRepo.Update(ctx, updatedData); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	if err := uc.tokenRepo.Delete(ctx, tokenData); err != nil {
		uc.logger.Warn("Failed to delete confirmation token", ports.F("error", err))
	}

	if err := uc.sendWelcomeEmail(ctx, subscription); err != nil {
		uc.logger.Warn("Failed to send welcome email", ports.F("error", err))
	}

	uc.logger.Debug("Subscription confirmed successfully",
		ports.F("email", subscription.Email),
		ports.F("city", subscription.City))
	return nil
}

func (uc *UseCase) Unsubscribe(ctx context.Context, params UnsubscribeParams) error {
	if params.Token == "" {
		return errors.NewValidationError("token is required")
	}

	uc.logger.Debug("Unsubscribing", ports.F("token", params.Token))

	tokenData, err := uc.tokenRepo.FindByToken(ctx, params.Token)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return errors.NewTokenError("invalid unsubscribe token")
		}
		return fmt.Errorf("find token: %w", err)
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return errors.NewTokenError("invalid unsubscribe token")
	}

	if tokenData.Type != "unsubscribe" {
		return errors.NewTokenError("invalid token type")
	}

	subscriptionData, err := uc.subscriptionRepo.FindByID(ctx, tokenData.SubscriptionID)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return errors.NewNotFoundError("subscription not found")
		}
		return fmt.Errorf("find subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Delete(ctx, subscriptionData); err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}

	if err := uc.tokenRepo.Delete(ctx, tokenData); err != nil {
		uc.logger.Warn("Failed to delete unsubscribe token", ports.F("error", err))
	}

	subscription := uc.convertFromPortsSubscription(subscriptionData)
	if err := uc.sendUnsubscribeConfirmationEmail(ctx, subscription); err != nil {
		uc.logger.Warn("Failed to send unsubscribe confirmation email", ports.F("error", err))
	}

	uc.logger.Debug("Unsubscribed successfully",
		ports.F("email", subscription.Email),
		ports.F("city", subscription.City))
	return nil
}

func (uc *UseCase) GetSubscriptionsForUpdates(ctx context.Context, frequency Frequency) ([]*Subscription, error) {
	if !frequency.IsValid() {
		return nil, errors.NewValidationError("invalid frequency")
	}

	subscriptionsData, err := uc.subscriptionRepo.GetConfirmedByFrequency(ctx, frequency.String())
	if err != nil {
		return nil, fmt.Errorf("get subscriptions for frequency %s: %w", frequency, err)
	}

	subscriptions := make([]*Subscription, len(subscriptionsData))
	for i, data := range subscriptionsData {
		subscriptions[i] = uc.convertFromPortsSubscription(data)
	}

	return subscriptions, nil
}

func (uc *UseCase) sendConfirmationEmail(ctx context.Context, subscription *Subscription) error {
	uc.logger.Debug("Creating confirmation token",
		ports.F("subscriptionID", subscription.ID))

	confirmToken, err := uc.tokenRepo.CreateConfirmationToken(ctx, subscription.ID, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("create confirmation token: %w", err)
	}

	emailParams := ports.EmailParams{
		To:      subscription.Email,
		Subject: "Confirm your weather subscription",
		Body:    uc.buildConfirmationEmailBody(subscription, confirmToken.Value),
		IsHTML:  true,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send confirmation email: %w", err)
	}

	return nil
}

func (uc *UseCase) sendWelcomeEmail(ctx context.Context, subscription *Subscription) error {
	unsubscribeToken, err := uc.tokenRepo.CreateUnsubscribeToken(ctx, subscription.ID, 365*24*time.Hour)
	if err != nil {
		uc.logger.Warn("Failed to create unsubscribe token", ports.F("error", err))
		return nil
	}

	emailParams := ports.EmailParams{
		To:      subscription.Email,
		Subject: "Welcome to Weather Updates!",
		Body:    uc.buildWelcomeEmailBody(subscription, unsubscribeToken.Value),
		IsHTML:  true,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send welcome email: %w", err)
	}

	return nil
}

func (uc *UseCase) sendUnsubscribeConfirmationEmail(ctx context.Context, subscription *Subscription) error {
	emailParams := ports.EmailParams{
		To:      subscription.Email,
		Subject: "You have been unsubscribed from weather updates",
		Body:    uc.buildUnsubscribeConfirmationBody(subscription),
		IsHTML:  true,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send unsubscribe confirmation email: %w", err)
	}

	return nil
}

func (uc *UseCase) convertToPortsSubscription(sub *Subscription) *ports.SubscriptionData {
	return &ports.SubscriptionData{
		ID:        sub.ID,
		Email:     sub.Email,
		City:      sub.City,
		Frequency: sub.Frequency.String(),
		Confirmed: sub.Confirmed,
		CreatedAt: sub.CreatedAt,
		UpdatedAt: sub.UpdatedAt,
	}
}

func (uc *UseCase) convertFromPortsSubscription(data *ports.SubscriptionData) *Subscription {
	return &Subscription{
		ID:        data.ID,
		Email:     data.Email,
		City:      data.City,
		Frequency: FrequencyFromString(data.Frequency),
		Confirmed: data.Confirmed,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}
}

func (uc *UseCase) buildConfirmationEmailBody(subscription *Subscription, token string) string {
	baseURL := uc.config.GetAppConfig().BaseURL
	confirmURL := fmt.Sprintf("%s/api/confirm/%s", baseURL, token)

	return fmt.Sprintf(`
		<h2>Confirm Your Weather Subscription</h2>
		<p>Hello!</p>
		<p>Thank you for subscribing to weather updates for <strong>%s</strong>.</p>
		<p>Please click the link below to confirm your subscription:</p>
		<p><a href="%s">Confirm Subscription</a></p>
		<p>If you didn't request this subscription, you can safely ignore this email.</p>
	`, subscription.City, confirmURL)
}

func (uc *UseCase) buildWelcomeEmailBody(subscription *Subscription, unsubscribeToken string) string {
	baseURL := uc.config.GetAppConfig().BaseURL
	unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", baseURL, unsubscribeToken)

	return fmt.Sprintf(`
		<h2>Welcome to Weather Updates!</h2>
		<p>Hello!</p>
		<p>Your subscription for <strong>%s</strong> weather updates has been confirmed.</p>
		<p>You will receive <strong>%s</strong> weather updates.</p>
		<p>If you wish to unsubscribe, click <a href="%s">here</a>.</p>
	`, subscription.City, subscription.Frequency, unsubscribeURL)
}

func (uc *UseCase) buildUnsubscribeConfirmationBody(subscription *Subscription) string {
	return fmt.Sprintf(`
		<h2>Unsubscribed Successfully</h2>
		<p>Hello!</p>
		<p>You have been successfully unsubscribed from weather updates for <strong>%s</strong>.</p>
		<p>We're sorry to see you go!</p>
	`, subscription.City)
}
