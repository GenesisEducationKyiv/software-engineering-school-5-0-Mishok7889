package infrastructure

import "weatherapi.app/internal/ports"

// Field represents a log field for structured logging
// This is a simple struct that doesn't depend on ports
type Field struct {
	Key   string
	Value interface{}
}

// F creates a log field for structured logging
// This function provides a consistent way to create log fields across all adapter layers
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// ToPortsFields converts infrastructure fields to ports fields for logger compatibility
func ToPortsFields(fields ...Field) []ports.Field {
	result := make([]ports.Field, len(fields))
	for i, field := range fields {
		result[i] = ports.Field{Key: field.Key, Value: field.Value}
	}
	return result
}
