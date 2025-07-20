package services

import (
	"context"

	"weatherapi.app/internal/ports"
)

// SubscriptionServiceAdapter adapts the subscription repository to the SubscriptionService port
type SubscriptionServiceAdapter struct {
	subscriptionRepo ports.SubscriptionRepository
	tokenRepo        ports.TokenRepository
}

// NewSubscriptionServiceAdapter creates a new subscription service adapter
func NewSubscriptionServiceAdapter(subscriptionRepo ports.SubscriptionRepository, tokenRepo ports.TokenRepository) ports.SubscriptionService {
	return &SubscriptionServiceAdapter{
		subscriptionRepo: subscriptionRepo,
		tokenRepo:        tokenRepo,
	}
}

// GetConfirmedSubscriptions returns all confirmed subscriptions for a given frequency
func (a *SubscriptionServiceAdapter) GetConfirmedSubscriptions(ctx context.Context, frequency string) ([]*ports.SubscriptionServiceData, error) {
	confirmed := true
	filter := ports.SubscriptionFilter{
		Frequency: &frequency,
		Confirmed: &confirmed,
	}

	subscriptions, err := a.subscriptionRepo.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return a.convertToServiceData(ctx, subscriptions)
}

// FindByID finds a subscription by ID
func (a *SubscriptionServiceAdapter) FindByID(ctx context.Context, id uint) (*ports.SubscriptionServiceData, error) {
	sub, err := a.subscriptionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get unsubscribe token
	token, err := a.tokenRepo.FindBySubscriptionID(ctx, sub.ID, "unsubscribe")
	if err != nil {
		return nil, err
	}

	return &ports.SubscriptionServiceData{
		ID:               sub.ID,
		Email:            sub.Email,
		City:             sub.City,
		Frequency:        sub.Frequency,
		Confirmed:        sub.Confirmed,
		UnsubscribeToken: token.Value,
	}, nil
}

// convertToServiceData converts repository data to service data
func (a *SubscriptionServiceAdapter) convertToServiceData(ctx context.Context, subscriptions []*ports.SubscriptionData) ([]*ports.SubscriptionServiceData, error) {
	result := make([]*ports.SubscriptionServiceData, 0, len(subscriptions))

	for _, sub := range subscriptions {
		// Get unsubscribe token for each subscription
		token, err := a.tokenRepo.FindBySubscriptionID(ctx, sub.ID, "unsubscribe")
		if err != nil {
			continue // Skip if token not found
		}

		result = append(result, &ports.SubscriptionServiceData{
			ID:               sub.ID,
			Email:            sub.Email,
			City:             sub.City,
			Frequency:        sub.Frequency,
			Confirmed:        sub.Confirmed,
			UnsubscribeToken: token.Value,
		})
	}

	return result, nil
}
