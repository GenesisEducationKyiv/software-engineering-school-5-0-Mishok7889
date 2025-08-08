package infrastructure

import "github.com/google/uuid"

// UUIDTokenGenerator generates tokens using UUID
type UUIDTokenGenerator struct{}

// NewUUIDTokenGenerator creates a new UUID token generator
func NewUUIDTokenGenerator() *UUIDTokenGenerator {
	return &UUIDTokenGenerator{}
}

// GenerateToken generates a new UUID token
func (g *UUIDTokenGenerator) GenerateToken() string {
	return uuid.New().String()
}
