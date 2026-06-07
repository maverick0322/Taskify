package util

import (
	"testing"

	"github.com/google/uuid"
)

func TestUUIDGenerator_Generate_ReturnsNonEmptyID(t *testing.T) {
	// Arrange
	generator := NewUUIDGenerator()

	// Act
	generatedID := generator.Generate()

	// Assert
	if generatedID == "" {
		t.Fatal("expected generated ID, got empty string")
	}
}

func TestUUIDGenerator_Generate_ReturnsValidUUID(t *testing.T) {
	// Arrange
	generator := NewUUIDGenerator()

	// Act
	generatedID := generator.Generate()
	parsedUUID, err := uuid.Parse(generatedID)

	// Assert
	if err != nil {
		t.Fatalf("expected valid UUID, got: %v", err)
	}
	if parsedUUID.String() != generatedID {
		t.Errorf("expected parsed UUID %s, got %s", generatedID, parsedUUID.String())
	}
}

func TestUUIDGenerator_GenerateConsecutiveIDs_ReturnsDifferentIDs(t *testing.T) {
	// Arrange
	generator := NewUUIDGenerator()

	// Act
	firstGeneratedID := generator.Generate()
	secondGeneratedID := generator.Generate()

	// Assert
	if firstGeneratedID == secondGeneratedID {
		t.Fatal("expected consecutive IDs to be different")
	}
}
