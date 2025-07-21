package subscription

import (
	"testing"
	"time"
)

func TestConstants(t *testing.T) {
	// Test token TTL constants
	if ConfirmationTokenTTL != 24*time.Hour {
		t.Errorf("ConfirmationTokenTTL = %v, want %v", ConfirmationTokenTTL, 24*time.Hour)
	}

	if UnsubscribeTokenTTL != 365*24*time.Hour {
		t.Errorf("UnsubscribeTokenTTL = %v, want %v", UnsubscribeTokenTTL, 365*24*time.Hour)
	}

	if SubscriptionTTL != 24*time.Hour {
		t.Errorf("SubscriptionTTL = %v, want %v", SubscriptionTTL, 24*time.Hour)
	}

	// Test error message constants are not empty
	errorConstants := []string{
		ErrEmailRequired,
		ErrEmailEmpty,
		ErrEmailInvalid,
		ErrCityRequired,
		ErrCityEmpty,
		ErrFrequencyInvalid,
		ErrFrequencyRequired,
		ErrTokenConfirmEmpty,
		ErrTokenUnsubEmpty,
		ErrTokenConfirmExpired,
		ErrTokenUnsubExpired,
		ErrTokenInvalidType,
		ErrSubscriptionExists,
		ErrSubscriptionConfirmed,
		ErrSubscriptionNotFound,
		ErrSubscriptionIDZero,
	}

	for _, errMsg := range errorConstants {
		if errMsg == "" {
			t.Errorf("Error constant should not be empty")
		}
	}

	// Test email template constants are not empty
	emailConstants := []string{
		EmailSubjectConfirmation,
		EmailSubjectWelcome,
		EmailSubjectUnsubscribe,
		EmailBodyConfirmation,
		EmailBodyWelcome,
		EmailBodyUnsubscribe,
	}

	for _, template := range emailConstants {
		if template == "" {
			t.Errorf("Email template constant should not be empty")
		}
	}

	// Test API path constants
	if APIPathConfirm != "/api/confirm/%s" {
		t.Errorf("APIPathConfirm = %v, want %v", APIPathConfirm, "/api/confirm/%s")
	}

	if APIPathUnsubscribe != "/api/unsubscribe/%s" {
		t.Errorf("APIPathUnsubscribe = %v, want %v", APIPathUnsubscribe, "/api/unsubscribe/%s")
	}
}
