package subscription

import (
	"context"
	"fmt"
	"time"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/core/token"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/validation"
)

type UseCase struct {
	subscriptionRepo ports.SubscriptionRepository
	tokenRepo        ports.TokenRepository
	tokenGenerator   ports.TokenGenerator
	emailProvider    ports.EmailProvider
	config           ports.ConfigProvider
	logger           ports.Logger
}

type UseCaseDependencies struct {
	SubscriptionRepo ports.SubscriptionRepository
	TokenRepo        ports.TokenRepository
	TokenGenerator   ports.TokenGenerator
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
		return nil, shared.NewValidationError(ErrRepoRequired)
	}
	if deps.TokenRepo == nil {
		return nil, shared.NewValidationError(ErrTokenRepoRequired)
	}
	if deps.TokenGenerator == nil {
		return nil, shared.NewValidationError(ErrGeneratorRequired)
	}
	if deps.EmailProvider == nil {
		return nil, shared.NewValidationError(ErrEmailProviderRequired)
	}
	if deps.Config == nil {
		return nil, shared.NewValidationError(ErrConfigRequired)
	}
	if deps.Logger == nil {
		return nil, shared.NewValidationError(ErrLoggerRequired)
	}

	return &UseCase{
		subscriptionRepo: deps.SubscriptionRepo,
		tokenRepo:        deps.TokenRepo,
		tokenGenerator:   deps.TokenGenerator,
		emailProvider:    deps.EmailProvider,
		config:           deps.Config,
		logger:           deps.Logger,
	}, nil
}

type CreateTokenParams struct {
	SubscriptionID uint
	TokenType      string
	ExpiresIn      time.Duration
}

func (uc *UseCase) validateSubscribeParams(params SubscribeParams) error {
	if !validation.IsNotEmpty(params.Email) {
		return shared.NewValidationError(ErrEmailRequired)
	}
	if !validation.IsValidEmail(params.Email) {
		return shared.NewValidationError(ErrEmailInvalid)
	}

	if !validation.IsNotEmpty(params.City) {
		return shared.NewValidationError(ErrCityRequired)
	}

	if !params.Frequency.IsValid() {
		return shared.NewValidationError(ErrFrequencyInvalid)
	}

	return nil
}

func (uc *UseCase) validateConfirmParams(params ConfirmParams) error {
	if !validation.IsNotEmpty(params.Token) {
		return shared.NewValidationError(ErrTokenConfirmEmpty)
	}
	return nil
}

func (uc *UseCase) validateUnsubscribeParams(params UnsubscribeParams) error {
	if !validation.IsNotEmpty(params.Token) {
		return shared.NewValidationError(ErrTokenUnsubEmpty)
	}
	return nil
}

func (uc *UseCase) createToken(ctx context.Context, params CreateTokenParams) (*ports.TokenData, error) {
	if params.SubscriptionID == 0 {
		return nil, shared.NewValidationError(ErrSubscriptionIDZero)
	}

	token := &ports.TokenData{
		Value:          uc.tokenGenerator.GenerateToken(),
		SubscriptionID: params.SubscriptionID,
		Type:           params.TokenType,
		ExpiresAt:      time.Now().Add(params.ExpiresIn),
		CreatedAt:      time.Now(),
	}

	if err := uc.tokenRepo.Save(ctx, token); err != nil {
		return nil, fmt.Errorf("save %s token: %w", params.TokenType, err)
	}

	return token, nil
}

func (uc *UseCase) Subscribe(ctx context.Context, params SubscribeParams) error {
	if err := uc.validateSubscribeParams(params); err != nil {
		return err
	}

	uc.logger.Debug("Processing subscription",
		ports.F("email", params.Email),
		ports.F("city", params.City),
		ports.F("frequency", params.Frequency.String()))

	existing, err := uc.subscriptionRepo.FindByEmail(ctx, params.Email, params.City)
	if err != nil && !shared.IsNotFoundError(err) && !ports.IsNotFoundError(err) {
		return fmt.Errorf("check existing subscription: %w", err)
	}

	if existing != nil {
		return uc.handleExistingSubscription(ctx, existing, params)
	}

	return uc.createNewSubscription(ctx, params)
}

func (uc *UseCase) handleExistingSubscription(ctx context.Context, existing *ports.SubscriptionData, params SubscribeParams) error {
	subscription := uc.convertFromPortsSubscription(existing)
	if subscription.IsConfirmed() {
		return shared.NewAlreadyExistsError(ErrSubscriptionExists)
	}

	if !subscription.IsExpired() {
		uc.logger.Debug("Updating existing unconfirmed subscription",
			ports.F("subscriptionID", existing.ID),
			ports.F("oldFrequency", existing.Frequency),
			ports.F("newFrequency", params.Frequency.String()))

		existing.Frequency = params.Frequency.String()
		existing.UpdatedAt = time.Now()

		if err := uc.subscriptionRepo.Update(ctx, existing); err != nil {
			return fmt.Errorf("update existing subscription: %w", err)
		}

		updatedSubscription := uc.convertFromPortsSubscription(existing)
		if err := uc.sendConfirmationEmail(ctx, updatedSubscription); err != nil {
			uc.logger.Error("Failed to send confirmation email for updated subscription",
				ports.F("error", err),
				ports.F("email", params.Email))
			return fmt.Errorf("send confirmation email: %w", err)
		}

		uc.logger.Debug("Existing subscription updated successfully",
			ports.F("email", params.Email),
			ports.F("city", params.City),
			ports.F("frequency", params.Frequency.String()))
		return nil
	}

	if err := uc.subscriptionRepo.Delete(ctx, existing); err != nil {
		uc.logger.Warn("Failed to delete expired subscription", ports.F("error", err))
	}

	return uc.createNewSubscription(ctx, params)
}

func (uc *UseCase) createNewSubscription(ctx context.Context, params SubscribeParams) error {
	subscription := NewSubscription(params.Email, params.City, params.Frequency)
	subscriptionData := uc.convertToPortsSubscription(subscription)
	if err := uc.subscriptionRepo.Save(ctx, subscriptionData); err != nil {
		return fmt.Errorf("save subscription: %w", err)
	}

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
	if err := uc.validateConfirmParams(params); err != nil {
		return err
	}

	uc.logger.Debug("Confirming subscription", ports.F("token", params.Token))

	tokenData, err := uc.tokenRepo.FindByToken(ctx, params.Token)
	if err != nil {
		if shared.IsNotFoundError(err) || ports.IsNotFoundError(err) {
			return shared.NewValidationError(ErrTokenConfirmExpired)
		}
		return fmt.Errorf("find token: %w", err)
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return shared.NewValidationError(ErrTokenConfirmExpired)
	}

	if tokenData.Type != token.TypeConfirmation.String() {
		return shared.NewValidationError(ErrTokenInvalidType)
	}

	subscriptionData, err := uc.subscriptionRepo.FindByID(ctx, tokenData.SubscriptionID)
	if err != nil {
		if shared.IsNotFoundError(err) || ports.IsNotFoundError(err) {
			return shared.NewNotFoundError(ErrSubscriptionNotFound)
		}
		return fmt.Errorf("find subscription: %w", err)
	}

	subscription := uc.convertFromPortsSubscription(subscriptionData)
	if subscription.IsConfirmed() {
		return shared.NewAlreadyExistsError(ErrSubscriptionConfirmed)
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
	if err := uc.validateUnsubscribeParams(params); err != nil {
		return err
	}

	uc.logger.Debug("Unsubscribing", ports.F("token", params.Token))

	tokenData, err := uc.tokenRepo.FindByToken(ctx, params.Token)
	if err != nil {
		if shared.IsNotFoundError(err) || ports.IsNotFoundError(err) {
			return shared.NewValidationError(ErrTokenUnsubExpired)
		}
		return fmt.Errorf("find token: %w", err)
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return shared.NewValidationError(ErrTokenUnsubExpired)
	}

	if tokenData.Type != token.TypeUnsubscribe.String() {
		return shared.NewValidationError(ErrTokenInvalidType)
	}

	subscriptionData, err := uc.subscriptionRepo.FindByID(ctx, tokenData.SubscriptionID)
	if err != nil {
		if shared.IsNotFoundError(err) || ports.IsNotFoundError(err) {
			return shared.NewNotFoundError(ErrSubscriptionNotFound)
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
		return nil, shared.NewValidationError(ErrFrequencyInvalid)
	}

	freqStr := frequency.String()
	confirmed := true
	filter := ports.SubscriptionFilter{
		Frequency: &freqStr,
		Confirmed: &confirmed,
	}

	subscriptionsData, err := uc.subscriptionRepo.Find(ctx, filter)
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

	confirmToken, err := uc.createToken(ctx, CreateTokenParams{
		SubscriptionID: subscription.ID,
		TokenType:      token.TypeConfirmation.String(),
		ExpiresIn:      ConfirmationTokenTTL,
	})
	if err != nil {
		return fmt.Errorf("create confirmation token: %w", err)
	}

	emailParams := ports.EmailParams{
		To:      subscription.Email,
		Subject: EmailSubjectConfirmation,
		Body:    uc.buildConfirmationEmailBody(subscription, confirmToken.Value),
		Format:  ports.FormatHTML,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send confirmation email: %w", err)
	}

	return nil
}

func (uc *UseCase) sendWelcomeEmail(ctx context.Context, subscription *Subscription) error {
	unsubscribeToken, err := uc.createToken(ctx, CreateTokenParams{
		SubscriptionID: subscription.ID,
		TokenType:      token.TypeUnsubscribe.String(),
		ExpiresIn:      UnsubscribeTokenTTL,
	})
	if err != nil {
		uc.logger.Warn("Failed to create unsubscribe token", ports.F("error", err))
		return nil
	}

	emailParams := ports.EmailParams{
		To:      subscription.Email,
		Subject: EmailSubjectWelcome,
		Body:    uc.buildWelcomeEmailBody(subscription, unsubscribeToken.Value),
		Format:  ports.FormatHTML,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send welcome email: %w", err)
	}

	return nil
}

func (uc *UseCase) sendUnsubscribeConfirmationEmail(ctx context.Context, subscription *Subscription) error {
	emailParams := ports.EmailParams{
		To:      subscription.Email,
		Subject: EmailSubjectUnsubscribe,
		Body:    uc.buildUnsubscribeConfirmationBody(subscription),
		Format:  ports.FormatHTML,
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
	confirmURL := fmt.Sprintf("%s"+APIPathConfirm, baseURL, token)

	return fmt.Sprintf(EmailBodyConfirmation, subscription.City, confirmURL)
}

func (uc *UseCase) buildWelcomeEmailBody(subscription *Subscription, unsubscribeToken string) string {
	baseURL := uc.config.GetAppConfig().BaseURL
	unsubscribeURL := fmt.Sprintf("%s"+APIPathUnsubscribe, baseURL, unsubscribeToken)

	return fmt.Sprintf(EmailBodyWelcome, subscription.City, subscription.Frequency, unsubscribeURL)
}

func (uc *UseCase) buildUnsubscribeConfirmationBody(subscription *Subscription) string {
	return fmt.Sprintf(EmailBodyUnsubscribe, subscription.City)
}
