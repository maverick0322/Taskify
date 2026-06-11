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

type fakePostgresCreditCardDatabase struct {
	execError         error
	queryError        error
	rowToReturn       pgx.Row
	rowsToReturn      pgx.Rows
	receivedSQL       string
	receivedArguments []interface{}
}

func (database *fakePostgresCreditCardDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresCreditCardDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowsToReturn, database.queryError
}

func (database *fakePostgresCreditCardDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowToReturn
}

type fakePostgresCreditCardRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresCreditCardRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	assignCreditCardScanValues(destinations, row.values)
	return nil
}

type fakePostgresCreditCardRows struct {
	rows        []fakePostgresCreditCardRow
	currentRow  int
	errToReturn error
	closed      bool
}

func (rows *fakePostgresCreditCardRows) Close() {
	rows.closed = true
}

func (rows *fakePostgresCreditCardRows) Err() error {
	return rows.errToReturn
}

func (rows *fakePostgresCreditCardRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakePostgresCreditCardRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakePostgresCreditCardRows) Next() bool {
	return rows.currentRow < len(rows.rows)
}

func (rows *fakePostgresCreditCardRows) Scan(destinations ...interface{}) error {
	row := rows.rows[rows.currentRow]
	rows.currentRow++
	return row.Scan(destinations...)
}

func (rows *fakePostgresCreditCardRows) Values() ([]interface{}, error) {
	return nil, nil
}

func (rows *fakePostgresCreditCardRows) RawValues() [][]byte {
	return nil
}

func (rows *fakePostgresCreditCardRows) Conn() *pgx.Conn {
	return nil
}

func TestPostgresCreditCardRepository_CreateValidCreditCard_ReturnsNil(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), createRepositoryCreditCard(t))

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != createCreditCardQuery {
		t.Errorf("expected create credit card query to be used")
	}
	if len(database.receivedArguments) != 11 {
		t.Errorf("expected eleven arguments, got %d", len(database.receivedArguments))
	}
}

func TestPostgresCreditCardRepository_CreateNilCreditCard_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	repository := &PostgresCreditCardRepository{database: &fakePostgresCreditCardDatabase{}, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), nil)

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_CreateDuplicateCreditCard_ReturnsErrCreditCardAlreadyExists(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{execError: &pgconn.PgError{Code: postgresUniqueViolationCode}}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), createRepositoryCreditCard(t))

	if !errors.Is(err, ports.ErrCreditCardAlreadyExists) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardAlreadyExists, err)
	}
}

func TestPostgresCreditCardRepository_CreateDatabaseFailure_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{execError: errors.New("database failure")}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Create(context.Background(), createRepositoryCreditCard(t))

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_GetByIDExistingCreditCard_ReturnsCreditCard(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{rowToReturn: fakePostgresCreditCardRow{values: validStoredCreditCardValues()}}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	creditCard, err := repository.GetByID(context.Background(), "credit-card-123")

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if creditCard.ID() != "credit-card-123" {
		t.Errorf("expected credit card ID credit-card-123, got %s", creditCard.ID())
	}
	if creditCard.Last4() != "1234" {
		t.Errorf("expected last4 1234, got %s", creditCard.Last4())
	}
}

func TestPostgresCreditCardRepository_GetByIDMissingCreditCard_ReturnsErrCreditCardNotFound(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{rowToReturn: fakePostgresCreditCardRow{err: pgx.ErrNoRows}}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByID(context.Background(), "credit-card-123")

	if !errors.Is(err, ports.ErrCreditCardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardNotFound, err)
	}
}

func TestPostgresCreditCardRepository_GetByIDCorruptedCreditCard_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	values := validStoredCreditCardValues()
	values[5] = 32
	database := &fakePostgresCreditCardDatabase{rowToReturn: fakePostgresCreditCardRow{values: values}}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByID(context.Background(), "credit-card-123")

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_GetByUserIDExistingCreditCards_ReturnsCreditCards(t *testing.T) {
	rows := &fakePostgresCreditCardRows{rows: []fakePostgresCreditCardRow{{values: validStoredCreditCardValues()}}}
	database := &fakePostgresCreditCardDatabase{rowsToReturn: rows}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	creditCards, err := repository.GetByUserID(context.Background(), "user-123")

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(creditCards) != 1 {
		t.Fatalf("expected one credit card, got %d", len(creditCards))
	}
	if database.receivedSQL != getCreditCardsByUserIDQuery {
		t.Errorf("expected user credit cards query to be used")
	}
	if len(database.receivedArguments) != 1 {
		t.Fatalf("expected one query argument, got %d", len(database.receivedArguments))
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
}

func TestPostgresCreditCardRepository_GetByUserIDQueryFailure_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{queryError: errors.New("query failure")}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByUserID(context.Background(), "user-123")

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_GetByUserIDScanFailure_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	values := validStoredCreditCardValues()
	values[8] = ""
	rows := &fakePostgresCreditCardRows{rows: []fakePostgresCreditCardRow{{values: values}}}
	database := &fakePostgresCreditCardDatabase{rowsToReturn: rows}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByUserID(context.Background(), "user-123")

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_GetByUserIDRowsError_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	rows := &fakePostgresCreditCardRows{errToReturn: errors.New("rows failure")}
	database := &fakePostgresCreditCardDatabase{rowsToReturn: rows}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	_, err := repository.GetByUserID(context.Background(), "user-123")

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_UpdateValidCreditCard_ReturnsNil(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Update(context.Background(), createRepositoryCreditCard(t))

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != updateCreditCardQuery {
		t.Errorf("expected update credit card query to be used")
	}
	if len(database.receivedArguments) != 9 {
		t.Errorf("expected nine arguments, got %d", len(database.receivedArguments))
	}
}

func TestPostgresCreditCardRepository_UpdateNilCreditCard_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	repository := &PostgresCreditCardRepository{database: &fakePostgresCreditCardDatabase{}, logger: &fakeRepositoryLogger{}}

	err := repository.Update(context.Background(), nil)

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_UpdateDatabaseFailure_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{execError: errors.New("database failure")}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Update(context.Background(), createRepositoryCreditCard(t))

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func TestPostgresCreditCardRepository_DeleteValidCreditCard_ReturnsNil(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Delete(context.Background(), "credit-card-123")

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != deleteCreditCardQuery {
		t.Errorf("expected delete credit card query to be used")
	}
}

func TestPostgresCreditCardRepository_DeleteDatabaseFailure_ReturnsErrCreditCardRepositoryUnavailable(t *testing.T) {
	database := &fakePostgresCreditCardDatabase{execError: errors.New("database failure")}
	repository := &PostgresCreditCardRepository{database: database, logger: &fakeRepositoryLogger{}}

	err := repository.Delete(context.Background(), "credit-card-123")

	if !errors.Is(err, ports.ErrCreditCardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardRepositoryUnavailable, err)
	}
}

func assignCreditCardScanValues(destinations []interface{}, values []interface{}) {
	for index, value := range values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *int:
			*destination = value.(int)
		case *int64:
			*destination = value.(int64)
		case *time.Time:
			*destination = value.(time.Time)
		}
	}
}

func validStoredCreditCardValues() []interface{} {
	return []interface{}{
		"credit-card-123",
		"user-123",
		"Clasica",
		"BBVA",
		"1234",
		15,
		5,
		int64(5000000),
		"from-blue-500 to-sky-400",
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-time.Hour),
	}
}

func createRepositoryCreditCard(t *testing.T) *domain.CreditCard {
	t.Helper()

	creditCard, err := domain.NewCreditCard(
		"credit-card-123",
		"user-123",
		"Clasica",
		"BBVA",
		"1234",
		15,
		5,
		5000000,
		"from-blue-500 to-sky-400",
	)
	if err != nil {
		t.Fatalf("expected credit card to be valid, got: %v", err)
	}

	return creditCard
}
