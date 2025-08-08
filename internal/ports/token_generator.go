package ports

// TokenGenerator defines the interface for generating tokens
type TokenGenerator interface {
	GenerateToken() string
}
