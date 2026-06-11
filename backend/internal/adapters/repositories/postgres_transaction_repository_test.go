package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type fakePostgresTransactionDatabase struct {
	execError         error
	queryError        error
	rowToReturn       pgx.Row
	rowsToReturn      pgx.Rows
	receivedSQL       string
	receivedArguments []interface{}
}

func (database *fakePostgresTransactionDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresTransactionDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowsToReturn, database.queryError
}

func (database *fakePostgresTransactionDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowToReturn
}

type fakePostgresTransactionRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresTransactionRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	assignTransactionScanValues(destinations, row.values)
	return nil
}

type fakePostgresTransactionRows struct {
	rows        []fakePostgresTransactionRow
	currentRow  int
	errToReturn error
	closed      bool
}

func (rows *fakePostgresTransactionRows) Close() {
	rows.closed = true
}

func (rows *fakePostgresTransactionRows) Err() error {
	return rows.errToReturn
}

func (rows *fakePostgresTransactionRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakePostgresTransactionRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakePostgresTransactionRows) Next() bool {
	return rows.currentRow < len(rows.rows)
}

func (rows *fakePostgresTransactionRows) Scan(destinations ...interface{}) error {
	row := rows.rows[rows.currentRow]
	rows.currentRow++
	return row.Scan(destinations...)
}

func (rows *fakePostgresTransactionRows) Values() ([]interface{}, error) {
	return nil, nil
}

func (rows *fakePostgresTransactionRows) RawValues() [][]byte {
	return nil
}

func (rows *fakePostgresTransactionRows) Conn() *pgx.Conn {
	return nil
}

func TestPostgresTransactionRepository_CreateValidTransaction_ReturnsNil(t *testing.T) {
	database := &fakePostgresTransactionDatabase{}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}
	transaction := createRepositoryTransaction(t, nil)

	err := repository.Create(context.Background(), transaction)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != createTransactionQuery {
		t.Errorf("expected create transaction query to be used")
	}
	if len(database.receivedArguments) != 12 {
		t.Errorf("expected twelve arguments, got %d", len(database.receivedArguments))
	}
}

func TestPostgresTransactionRepository_CreateNilTransaction_ReturnsErrTransactionRepositoryUnavailable(t *testing.T) {
	repository := &PostgresTransactionRepository{database: &fakePostgresTransactionDatabase{}, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), nil)

	if !errors.Is(err, ports.ErrTransactionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionRepositoryUnavailable, err)
	}
}

func TestPostgresTransactionRepository_CreateDuplicateTransaction_ReturnsErrTransactionAlreadyExists(t *testing.T) {
	database := &fakePostgresTransactionDatabase{execError: &pgconn.PgError{Code: postgresUniqueViolationCode}}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), createRepositoryTransaction(t, nil))

	if !errors.Is(err, ports.ErrTransactionAlreadyExists) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionAlreadyExists, err)
	}
}

func TestPostgresTransactionRepository_CreateWithMSI_StoresMSIValue(t *testing.T) {
	msi := 6
	database := &fakePostgresTransactionDatabase{}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), createRepositoryTransaction(t, &msi))

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedArguments[9] != 6 {
		t.Errorf("expected MSI argument 6, got %v", database.receivedArguments[9])
	}
}

func TestPostgresTransactionRepository_GetByIDExistingTransaction_ReturnsTransaction(t *testing.T) {
	database := &fakePostgresTransactionDatabase{rowToReturn: fakePostgresTransactionRow{values: validStoredTransactionValues(nil)}}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	transaction, err := repository.GetByID(context.Background(), "transaction-123")

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if transaction.ID() != "transaction-123" {
		t.Errorf("expected transaction ID transaction-123, got %s", transaction.ID())
	}
	if transaction.AmountCents() != 12500 {
		t.Errorf("expected amount cents 12500, got %d", transaction.AmountCents())
	}
}

func TestPostgresTransactionRepository_GetByIDMissingTransaction_ReturnsErrTransactionNotFound(t *testing.T) {
	database := &fakePostgresTransactionDatabase{rowToReturn: fakePostgresTransactionRow{err: pgx.ErrNoRows}}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByID(context.Background(), "transaction-123")

	if !errors.Is(err, ports.ErrTransactionNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionNotFound, err)
	}
}

func TestPostgresTransactionRepository_GetByIDCorruptedTransaction_ReturnsErrTransactionRepositoryUnavailable(t *testing.T) {
	values := validStoredTransactionValues(nil)
	values[3] = "TRANSFER"
	database := &fakePostgresTransactionDatabase{rowToReturn: fakePostgresTransactionRow{values: values}}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByID(context.Background(), "transaction-123")

	if !errors.Is(err, ports.ErrTransactionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionRepositoryUnavailable, err)
	}
}

func TestPostgresTransactionRepository_GetByUserIDWithoutFilter_UsesBaseQuery(t *testing.T) {
	rows := &fakePostgresTransactionRows{rows: []fakePostgresTransactionRow{{values: validStoredTransactionValues(nil)}}}
	database := &fakePostgresTransactionDatabase{rowsToReturn: rows}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	transactions, err := repository.GetByUserID(context.Background(), "user-123", ports.TransactionDateFilter{})

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected one transaction, got %d", len(transactions))
	}
	if database.receivedSQL != getTransactionsByUserIDQuery {
		t.Errorf("expected base user transaction query to be used")
	}
	if len(database.receivedArguments) != 1 {
		t.Fatalf("expected one query argument, got %d", len(database.receivedArguments))
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
}

func TestPostgresTransactionRepository_GetByUserIDWithDateRange_UsesRangeQuery(t *testing.T) {
	startDate := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	rows := &fakePostgresTransactionRows{rows: []fakePostgresTransactionRow{{values: validStoredTransactionValues(nil)}}}
	database := &fakePostgresTransactionDatabase{rowsToReturn: rows}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByUserID(context.Background(), "user-123", ports.TransactionDateFilter{From: &startDate, To: &endDate})

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != getTransactionsByUserIDRangeQuery {
		t.Errorf("expected date range transaction query to be used")
	}
	if len(database.receivedArguments) != 3 {
		t.Fatalf("expected three query arguments, got %d", len(database.receivedArguments))
	}
	if !database.receivedArguments[1].(time.Time).Equal(startDate) {
		t.Errorf("expected start date argument %v, got %v", startDate, database.receivedArguments[1])
	}
	if !database.receivedArguments[2].(time.Time).Equal(endDate) {
		t.Errorf("expected end date argument %v, got %v", endDate, database.receivedArguments[2])
	}
}

func TestPostgresTransactionRepository_GetByUserIDQueryFailure_ReturnsErrTransactionRepositoryUnavailable(t *testing.T) {
	database := &fakePostgresTransactionDatabase{queryError: errors.New("query failure")}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByUserID(context.Background(), "user-123", ports.TransactionDateFilter{})

	if !errors.Is(err, ports.ErrTransactionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionRepositoryUnavailable, err)
	}
}

func TestPostgresTransactionRepository_GetByUserIDRowsError_ReturnsErrTransactionRepositoryUnavailable(t *testing.T) {
	rows := &fakePostgresTransactionRows{errToReturn: errors.New("rows failure")}
	database := &fakePostgresTransactionDatabase{rowsToReturn: rows}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByUserID(context.Background(), "user-123", ports.TransactionDateFilter{})

	if !errors.Is(err, ports.ErrTransactionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionRepositoryUnavailable, err)
	}
}

func TestPostgresTransactionRepository_UpdateValidTransaction_ReturnsNil(t *testing.T) {
	database := &fakePostgresTransactionDatabase{}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Update(context.Background(), createRepositoryTransaction(t, nil))

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != updateTransactionQuery {
		t.Errorf("expected update transaction query to be used")
	}
}

func TestPostgresTransactionRepository_UpdateNilTransaction_ReturnsErrTransactionRepositoryUnavailable(t *testing.T) {
	repository := &PostgresTransactionRepository{database: &fakePostgresTransactionDatabase{}, logger: &fakeRepositoryLogger{}}

	err := repository.Update(context.Background(), nil)

	if !errors.Is(err, ports.ErrTransactionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionRepositoryUnavailable, err)
	}
}

func TestPostgresTransactionRepository_DeleteValidTransaction_ReturnsNil(t *testing.T) {
	database := &fakePostgresTransactionDatabase{}
	repository := &PostgresTransactionRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Delete(context.Background(), "transaction-123")

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != deleteTransactionQuery {
		t.Errorf("expected delete transaction query to be used")
	}
}

func assignTransactionScanValues(destinations []interface{}, values []interface{}) {
	for index, value := range values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *int64:
			*destination = value.(int64)
		case **string:
			if value == nil {
				*destination = nil
				continue
			}
			storedValue := value.(string)
			*destination = &storedValue
		case **int:
			if value == nil {
				*destination = nil
				continue
			}
			storedValue := value.(int)
			*destination = &storedValue
		case *time.Time:
			*destination = value.(time.Time)
		}
	}
}

func validStoredTransactionValues(msi *int) []interface{} {
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := time.Now().Add(-time.Hour)
	var storedMSI interface{}
	if msi != nil {
		storedMSI = *msi
	}

	return []interface{}{
		"transaction-123",
		"user-123",
		nil,
		string(domain.TransactionTypeExpense),
		"CFE - Luz",
		"Servicios",
		int64(12500),
		time.Now(),
		string(domain.TransactionStatusPaid),
		storedMSI,
		createdAt,
		updatedAt,
	}
}

func createRepositoryTransaction(t *testing.T, msi *int) *domain.Transaction {
	t.Helper()

	transaction, err := domain.NewTransaction(
		"transaction-123",
		"user-123",
		domain.TransactionTypeExpense,
		"CFE - Luz",
		"Servicios",
		12500,
		time.Now(),
		domain.TransactionStatusPaid,
		msi,
		nil,
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}
