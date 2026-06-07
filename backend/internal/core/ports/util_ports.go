package ports

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

// TokenGenerator abstracts the session token creation (e.g., JWT).
type TokenGenerator interface {
	GenerateToken(userID string) (string, error)
}

// IDGenerator abstracts the creation of unique identifiers (e.g., UUIDv4, ULID).
type IDGenerator interface {
	Generate() string
}
