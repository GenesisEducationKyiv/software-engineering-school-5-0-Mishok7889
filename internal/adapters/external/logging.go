package external

import "weatherapi.app/internal/ports"

// logField creates a log field for structured logging within external adapters
// This is a local helper that creates ports.Field without using ports.F() utility
func logField(key string, value any) ports.Field {
	return ports.Field{Key: key, Value: value}
}
