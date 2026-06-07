package util

import (
	"github.com/google/uuid"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// UUIDGenerator creates opaque identifiers at the infrastructure edge.
type UUIDGenerator struct{}

func NewUUIDGenerator() ports.IDGenerator {
	return &UUIDGenerator{}
}

func (generator *UUIDGenerator) Generate() string {
	return uuid.NewString()
}
