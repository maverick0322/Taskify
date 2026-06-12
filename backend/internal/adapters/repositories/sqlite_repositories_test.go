package repositories

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	_ "modernc.org/sqlite"
)

func TestSQLiteRepositories_PersistAndQueryLocalFirstData(t *testing.T) {
	// Arrange
	ctx := context.Background()
	database := openTestSQLiteDatabase(t)
	logger := &fakeRepositoryLogger{}

	userRepository := NewSQLiteUserRepository(database, logger)
	sessionRepository := NewSQLiteSessionRepository(database, logger)
	boardRepository := NewSQLiteBoardRepository(database, logger)
	columnRepository := NewSQLiteColumnRepository(database, logger)
	taskRepository := NewSQLiteTaskRepository(database, logger)
	creditCardRepository := NewSQLiteCreditCardRepository(database, logger)
	transactionRepository := NewSQLiteTransactionRepository(database, logger)

	profile, err := domain.NewUserProfile("Erick", "Lara", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}
	user, err := domain.NewUser("user-1", "erick@example.com", "hashed-password-value", profile)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Act + Assert: auth data
	if err := userRepository.Save(ctx, user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}
	if err := userRepository.Save(ctx, user); !errors.Is(err, ports.ErrUserAlreadyExists) {
		t.Fatalf("expected duplicate user error, got %v", err)
	}
	storedUser, err := userRepository.GetByEmail(ctx, "erick@example.com")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if storedUser.ID() != user.ID() {
		t.Fatalf("expected user ID %q, got %q", user.ID(), storedUser.ID())
	}

	refreshToken, err := domain.NewRefreshToken("session-1", user.ID(), "hash-1", time.Now().Add(24*time.Hour), false)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}
	rotatedRefreshToken, err := domain.NewRefreshToken("session-2", user.ID(), "hash-2", time.Now().Add(48*time.Hour), false)
	if err != nil {
		t.Fatalf("failed to create rotated refresh token: %v", err)
	}
	if err := sessionRepository.Save(ctx, refreshToken); err != nil {
		t.Fatalf("failed to save refresh token: %v", err)
	}
	if err := sessionRepository.Rotate(ctx, refreshToken.ID(), rotatedRefreshToken); err != nil {
		t.Fatalf("failed to rotate refresh token: %v", err)
	}
	revokedToken, err := sessionRepository.GetByTokenHash(ctx, refreshToken.TokenHash())
	if err != nil {
		t.Fatalf("failed to get revoked refresh token: %v", err)
	}
	if !revokedToken.IsRevoked() {
		t.Fatal("expected original refresh token to be revoked")
	}

	// Act + Assert: Kanban and tasks
	board, err := domain.NewBoard("board-1", user.ID(), "Trabajo")
	if err != nil {
		t.Fatalf("failed to create board: %v", err)
	}
	if err := boardRepository.Save(ctx, board); err != nil {
		t.Fatalf("failed to save board: %v", err)
	}
	column, err := domain.NewColumn("column-1", board.ID(), "Por hacer", 0)
	if err != nil {
		t.Fatalf("failed to create column: %v", err)
	}
	if err := columnRepository.Save(ctx, column); err != nil {
		t.Fatalf("failed to save column: %v", err)
	}
	if err := column.ChangePosition(1); err != nil {
		t.Fatalf("failed to move column: %v", err)
	}
	if err := columnRepository.UpdatePositions(ctx, []*domain.Column{column}); err != nil {
		t.Fatalf("failed to update column positions: %v", err)
	}

	boardID := board.ID()
	dueDate := time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)
	task, err := domain.NewTask("task-1", user.ID(), &boardID, "Preparar reporte", "Histórico", domain.TaskStatusTodo, domain.TaskPriorityHigh, dueDate)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if err := taskRepository.Save(ctx, task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}
	storedTask, err := taskRepository.GetByID(ctx, task.ID())
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if !storedTask.DueDate().Equal(dueDate) {
		t.Fatalf("expected due date %v, got %v", dueDate, storedTask.DueDate())
	}

	// Act + Assert: financial data and date filters
	creditCard, err := domain.NewCreditCard("card-1", user.ID(), "Clásica", "BBVA", "1234", 10, 20, 5000000, "from-blue-600 to-sky-500")
	if err != nil {
		t.Fatalf("failed to create credit card: %v", err)
	}
	if err := creditCardRepository.Create(ctx, creditCard); err != nil {
		t.Fatalf("failed to save credit card: %v", err)
	}

	msi := 3
	transactionDate := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	creditCardID := creditCard.ID()
	transaction, err := domain.NewTransaction("transaction-1", user.ID(), domain.TransactionTypeExpense, "Laptop", "Equipo", 120000, transactionDate, domain.TransactionStatusPending, &msi, &creditCardID)
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}
	if err := transactionRepository.Create(ctx, transaction); err != nil {
		t.Fatalf("failed to save transaction: %v", err)
	}
	transactions, err := transactionRepository.GetByUserID(ctx, user.ID(), ports.TransactionDateFilter{
		From: ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
		To:   ptrTime(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("failed to query transactions by date: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}

	if err := creditCardRepository.Delete(ctx, creditCard.ID()); err != nil {
		t.Fatalf("failed to delete credit card: %v", err)
	}
	transactionAfterCardDelete, err := transactionRepository.GetByID(ctx, transaction.ID())
	if err != nil {
		t.Fatalf("failed to get transaction after deleting card: %v", err)
	}
	if transactionAfterCardDelete.CreditCardID() != nil {
		t.Fatal("expected transaction credit card id to be null after card delete")
	}
}

func openTestSQLiteDatabase(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	schemaPath := filepath.Join("..", "..", "..", "..", "scripts", "init_sqlite.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read sqlite schema: %v", err)
	}
	if _, err := database.Exec(string(schema)); err != nil {
		t.Fatalf("failed to initialize sqlite schema: %v", err)
	}

	return database
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
