package database

// Error message constants for subscription repository
const (
	// Validation error messages
	ErrSubscriptionNil             = "subscription cannot be nil"
	ErrSubscriptionIDZero          = "subscription ID cannot be zero"
	ErrSubscriptionIDZeroForUpdate = "subscription ID cannot be zero for update"
	ErrSubscriptionIDZeroForDelete = "subscription ID cannot be zero for delete"
	ErrEmailEmpty                  = "email cannot be empty"
	ErrCityEmpty                   = "city cannot be empty"
	ErrFrequencyEmpty              = "frequency cannot be empty"
	ErrFilterCriteriaRequired      = "at least one filter criterion must be provided"

	// Operation error messages
	ErrFailedToSave             = "failed to save subscription"
	ErrFailedToFindByID         = "failed to find subscription by ID"
	ErrFailedToFind             = "failed to find subscription"
	ErrFailedToUpdate           = "failed to update subscription"
	ErrFailedToDelete           = "failed to delete subscription"
	ErrFailedToCountByFrequency = "failed to count subscriptions by frequency"
	ErrFailedToCountConfirmed   = "failed to count confirmed subscriptions"

	// Not found error messages
	ErrSubscriptionNotFound = "subscription not found"
)
