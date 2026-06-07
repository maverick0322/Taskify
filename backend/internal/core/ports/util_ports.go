package ports

import "time"

// Logger defines the contract for our application logging.
// We strictly use Info, Warn, and Error to prevent log noise.
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// PasswordHasher abstracts the cryptography implementation (e.g., bcrypt, argon2).
type PasswordHasher interface {
	Hash(plainPassword string) (string, error)
	Compare(plainPassword, hashedPassword string) error
}

// TokenPair keeps access and refresh credentials coupled at the port boundary.
type TokenPair struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
}

// TokenGenerator abstracts session token creation without exposing JWT details to the core.
type TokenGenerator interface {
	GenerateTokenPair(userID string) (TokenPair, error)
}

// TokenValidator abstracts access-token verification for inbound adapters.
type TokenValidator interface {
	ValidateToken(token string) (string, error)
}

// IDGenerator abstracts the creation of unique identifiers (e.g., UUIDv4, ULID).
type IDGenerator interface {
	Generate() string
}
