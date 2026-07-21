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
	creditCard, err := domain.NewCreditCard("card-1", user.ID(), "Clásica", "BBVA", "1234", 10, 20, 5000000, "from-blue-600 to-sky-500", domain.CreditCardNetworkVisa)
	if err != nil {
		t.Fatalf("failed to create credit card: %v", err)
	}
	if err := creditCardRepository.Create(ctx, creditCard); err != nil {
		t.Fatalf("failed to save credit card: %v", err)
	}

	msi := 3
	transactionDate := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	creditCardID := creditCard.ID()
	transaction, err := domain.NewTransaction("transaction-1", user.ID(), domain.TransactionTypeExpense, "Laptop", "Equipo", 120000, transactionDate, domain.TransactionStatusPending, &msi, &creditCardID, domain.TransactionRecurrenceOnce, nil)
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

func TestSQLiteCreditCardStatementRepository_ApplyPaymentCommitsAccountingAndAllocation(t *testing.T) {
	ctx := context.Background()
	database := openTestSQLiteDatabase(t)
	logger := &fakeRepositoryLogger{}
	userRepository := NewSQLiteUserRepository(database, logger)
	cardRepository := NewSQLiteCreditCardRepository(database, logger)
	accountRepository := NewSQLiteFinancialAccountRepository(database, logger)
	transactionRepository := NewSQLiteTransactionRepository(database, logger)
	statementRepository := NewSQLiteCreditCardStatementRepository(database, logger)

	profile, _ := domain.NewUserProfile("Erick", "Lara", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))
	user, _ := domain.NewUser("pay-user", "pay@example.com", "hashed-password-value", profile)
	if err := userRepository.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	card, _ := domain.NewCreditCard("pay-card", user.ID(), "Joy", "Joy", "1234", 5, 25, 500000, "#222222", domain.CreditCardNetworkVisa)
	if err := cardRepository.Create(ctx, card); err != nil {
		t.Fatal(err)
	}
	last4 := "7178"
	source, _ := domain.NewFinancialAccount("pay-source", user.ID(), domain.FinancialAccountTypeDebitCard, "Débito", "BBVA", &last4, 100000, 100000, nil, nil, nil, "#111111", domain.CreditCardNetworkVisa)
	if err := accountRepository.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`UPDATE financial_accounts SET current_balance_cents = 30000 WHERE id = ?`, card.ID()); err != nil {
		t.Fatal(err)
	}

	cardID := card.ID()
	purchaseDate := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	purchase, _ := domain.NewTransaction("pay-purchase", user.ID(), domain.TransactionTypeExpense, "Compra", "Otros", 30000, purchaseDate, domain.TransactionStatusCompleted, nil, &cardID, domain.TransactionRecurrenceOnce, nil)
	purchase.SetAccountingDetails(&cardID, nil, nil, nil)
	purchase.SetCreditCardID(&cardID)
	if err := transactionRepository.Create(ctx, purchase); err != nil {
		t.Fatal(err)
	}

	statement := ports.CreditCardStatement{ID: "statement-1", UserID: user.ID(), CreditAccountID: card.ID(), CycleStart: time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC), CycleEnd: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC), PaymentDueDate: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC), StatementAmountCents: 30000, Status: "DUE", CreatedAt: purchaseDate, UpdatedAt: purchaseDate}
	item := ports.CreditCardStatementItem{ID: "item-1", UserID: user.ID(), StatementID: statement.ID, TransactionID: purchase.ID(), AmountCents: 30000, CreatedAt: purchaseDate, UpdatedAt: purchaseDate}
	if err := statementRepository.Reconcile(ctx, user.ID(), card.ID(), []ports.CreditCardStatement{statement}, []ports.CreditCardStatementItem{item}, nil); err != nil {
		t.Fatal(err)
	}

	paymentDate := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	payment, _ := domain.NewTransaction("payment-1", user.ID(), domain.TransactionTypeDebtPayment, "Pago Joy", "Tarjeta de crédito", 30000, paymentDate, domain.TransactionStatusCompleted, nil, &cardID, domain.TransactionRecurrenceOnce, nil)
	payment.SetAccountingDetails(ptrString(source.ID()), &cardID, nil, nil)
	payment.SetCreditCardID(&cardID)
	statement.PaidAmountCents = 30000
	statement.Status = "PAID"
	statement.UpdatedAt = paymentDate
	allocation := ports.CreditCardPaymentAllocation{ID: "allocation-1", UserID: user.ID(), StatementID: statement.ID, PaymentTransactionID: payment.ID(), AmountCents: 30000, CreatedAt: paymentDate, UpdatedAt: paymentDate}
	deltas := []ports.AccountBalanceDelta{{AccountID: source.ID(), DeltaCents: -30000}, {AccountID: card.ID(), DeltaCents: -30000}}
	entries := []ports.LedgerEntry{{ID: "ledger-source", UserID: user.ID(), AccountID: source.ID(), TransactionID: payment.ID(), AmountCents: -30000, EntryType: string(domain.TransactionTypeDebtPayment), CreatedAt: paymentDate, UpdatedAt: paymentDate}, {ID: "ledger-card", UserID: user.ID(), AccountID: card.ID(), TransactionID: payment.ID(), AmountCents: -30000, EntryType: string(domain.TransactionTypeDebtPayment), CreatedAt: paymentDate, UpdatedAt: paymentDate}}

	if err := statementRepository.ApplyPayment(ctx, payment, deltas, entries, []ports.CreditCardPaymentAllocation{allocation}, []ports.CreditCardStatement{statement}); err != nil {
		t.Fatalf("expected atomic payment to succeed, got %v", err)
	}
	assertSQLiteInt64(t, database, `SELECT current_balance_cents FROM financial_accounts WHERE id = ?`, source.ID(), 70000)
	assertSQLiteInt64(t, database, `SELECT current_balance_cents FROM financial_accounts WHERE id = ?`, card.ID(), 0)
	assertSQLiteInt64(t, database, `SELECT COUNT(*) FROM ledger_entries WHERE transaction_id = ?`, payment.ID(), 2)
	assertSQLiteInt64(t, database, `SELECT amount_cents FROM credit_card_payment_allocations WHERE id = ?`, allocation.ID, 30000)
	assertSQLiteInt64(t, database, `SELECT paid_amount_cents FROM credit_card_statements WHERE id = ?`, statement.ID, 30000)
}

func ptrString(value string) *string { return &value }

func assertSQLiteInt64(t *testing.T, database *sql.DB, query string, argument interface{}, expected int64) {
	t.Helper()
	var actual int64
	if err := database.QueryRow(query, argument).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func TestSQLiteUserRepository_UpsertPreservesAvatarLocalPathAndRefreshesRemoteFields(t *testing.T) {
	ctx := context.Background()
	database := openTestSQLiteDatabase(t)
	logger := &fakeRepositoryLogger{}
	userRepository := NewSQLiteUserRepository(database, logger)

	profile, err := domain.NewUserProfile("Jane", "Doe", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to create initial profile: %v", err)
	}
	user, err := domain.NewUser("user-1", "jane@example.com", "hashed-password-value", profile)
	if err != nil {
		t.Fatalf("failed to create initial user: %v", err)
	}
	user.UpdateAvatar("C:/avatars/jane.png", "")
	if err := userRepository.Save(ctx, user); err != nil {
		t.Fatalf("failed to save initial user: %v", err)
	}

	updatedProfile, err := domain.NewUserProfile("Jane", "Doe", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to create updated profile: %v", err)
	}
	updatedUser, err := domain.NewUser("user-1", "jane@example.com", "new-hash-value", updatedProfile)
	if err != nil {
		t.Fatalf("failed to create updated user: %v", err)
	}
	updatedUser.UpdateAvatar("", "https://cdn.example.com/jane.png")
	if err := userRepository.Upsert(ctx, updatedUser); err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}

	storedUser, err := userRepository.GetByEmail(ctx, "jane@example.com")
	if err != nil {
		t.Fatalf("failed to retrieve upserted user: %v", err)
	}
	if storedUser.AvatarLocalPath() != "C:/avatars/jane.png" {
		t.Fatalf("expected local avatar path to be preserved, got %q", storedUser.AvatarLocalPath())
	}
	if storedUser.AvatarURL() != "https://cdn.example.com/jane.png" {
		t.Fatalf("expected avatar url to be refreshed, got %q", storedUser.AvatarURL())
	}
	if storedUser.PasswordHash() != "new-hash-value" {
		t.Fatalf("expected password hash to be refreshed, got %q", storedUser.PasswordHash())
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
