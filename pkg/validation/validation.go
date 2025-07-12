package validation

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

// IsNotEmpty checks if string is not empty after trimming
func IsNotEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

// IsValidFrequency validates subscription frequency
func IsValidFrequency(frequency string) bool {
	return frequency == "hourly" || frequency == "daily"
}

// TrimAndValidate trims string and validates it's not empty
func TrimAndValidate(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	return trimmed, trimmed != ""
}
