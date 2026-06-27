package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type transactionService struct {
	transactionRepository ports.TransactionRepository
	accountRepository     ports.FinancialAccountRepository
	idGenerator           ports.IDGenerator
	logger                ports.Logger
}

func NewTransactionService(
	transactionRepository ports.TransactionRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
	accountRepository ...ports.FinancialAccountRepository,
) ports.TransactionUseCase {
	var financialAccountRepository ports.FinancialAccountRepository
	if len(accountRepository) > 0 {
		financialAccountRepository = accountRepository[0]
	}
	return &transactionService{
		transactionRepository: transactionRepository,
		accountRepository:     financialAccountRepository,
		idGenerator:           idGenerator,
		logger:                logger,
	}
}

func (service *transactionService) CreateTransaction(
	ctx context.Context,
	userID string,
	transactionType domain.TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status domain.TransactionStatus,
	msi *int,
	creditCardID *string,
	recurrence domain.TransactionRecurrence,
	recurrenceLimit *int,
	paymentAccountID *string,
	paidInstallments ...int,
) (*domain.Transaction, error) {
	historicalInstallments, err := normalizePaidInstallments(msi, paidInstallments...)
	if err != nil {
		return nil, err
	}
	if paymentAccountID == nil {
		paymentAccountID = creditCardID
	}
	transactionID := service.idGenerator.Generate()
	transaction, err := domain.NewTransaction(transactionID, userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID, recurrence, recurrenceLimit)
	if err != nil {
		return nil, err
	}
	transaction.SetAccountingDetails(paymentAccountID, nil, nil, nil)

	if paymentAccountID != nil && service.accountRepository != nil {
		createdTransactions, deltas, ledgerEntries, err := service.prepareAccountedTransactions(ctx, userID, transaction, msi, historicalInstallments)
		if err != nil {
			return nil, err
		}
		if err := service.transactionRepository.CreateManyWithLedger(ctx, createdTransactions, deltas, ledgerEntries); err != nil {
			service.logger.Error("failed to create accounted transaction", "userID", userID, "transactionID", transactionID, "error", err)
			return nil, ErrInternalProcessing
		}
		return transaction, nil
	}

	if err := service.transactionRepository.Create(ctx, transaction); err != nil {
		service.logger.Error("failed to create transaction", "userID", userID, "transactionID", transactionID, "error", err)
		return nil, ErrInternalProcessing
	}

	return transaction, nil
}

func (service *transactionService) prepareAccountedTransactions(ctx context.Context, userID string, transaction *domain.Transaction, msi *int, paidInstallments int) ([]*domain.Transaction, []ports.AccountBalanceDelta, []ports.LedgerEntry, error) {
	accountID := transaction.PaymentAccountID()
	if accountID == nil {
		return []*domain.Transaction{transaction}, nil, nil, nil
	}
	account, err := service.accountRepository.GetByID(ctx, *accountID)
	if err != nil {
		return nil, nil, nil, err
	}
	if account.UserID() != userID {
		return nil, nil, nil, ports.ErrFinancialAccountNotFound
	}
	if account.Type() == domain.FinancialAccountTypeCreditCard {
		transaction.SetCreditCardID(accountID)
	}

	transactions := []*domain.Transaction{transaction}
	if transaction.Type() == domain.TransactionTypeExpense && account.Type() == domain.FinancialAccountTypeCreditCard && msi != nil && *msi > 1 {
		transactions = service.projectMSITransactions(userID, transaction, *msi, paidInstallments)
	}

	delta := int64(0)
	for _, currentTransaction := range transactions {
		if currentTransaction.IsHistorical() {
			continue
		}
		delta += transactionDeltaForAccount(currentTransaction.Type(), account.Type(), currentTransaction.AmountCents())
	}
	if err := account.ApplyDelta(delta); err != nil {
		return nil, nil, nil, err
	}

	deltas := []ports.AccountBalanceDelta{{AccountID: *accountID, DeltaCents: delta}}
	entries := make([]ports.LedgerEntry, 0, len(transactions))
	for _, currentTransaction := range transactions {
		if currentTransaction.IsHistorical() {
			continue
		}
		now := time.Now()
		entries = append(entries, ports.LedgerEntry{
			ID:            service.idGenerator.Generate(),
			UserID:        userID,
			AccountID:     *accountID,
			TransactionID: currentTransaction.ID(),
			AmountCents:   transactionDeltaForAccount(currentTransaction.Type(), account.Type(), currentTransaction.AmountCents()),
			EntryType:     string(currentTransaction.Type()),
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return transactions, deltas, entries, nil
}

func (service *transactionService) projectMSITransactions(userID string, original *domain.Transaction, months, paidInstallments int) []*domain.Transaction {
	installments := make([]*domain.Transaction, 0, months)
	baseAmount := original.AmountCents() / int64(months)
	remainder := original.AmountCents() % int64(months)
	originalDate := original.Date().AddDate(0, -paidInstallments, 0)
	for index := 1; index <= months; index++ {
		amount := baseAmount
		if index == 1 {
			amount += remainder
		}
		transactionID := original.ID()
		if index > 1 {
			transactionID = service.idGenerator.Generate()
		}
		installmentDate := originalDate.AddDate(0, index-1, 0)
		status := domain.TransactionStatusPending
		isHistorical := false
		if index <= paidInstallments {
			status = domain.TransactionStatusPaid
			isHistorical = true
		}
		installment, _ := domain.NewTransaction(transactionID, userID, domain.TransactionTypeExpense, fmt.Sprintf("%s %d/%d", original.Concept(), index, months), original.Category(), amount, installmentDate, status, &months, original.CreditCardID(), domain.TransactionRecurrenceOnce, nil)
		installment.SetAccountingDetails(original.PaymentAccountID(), nil, &index, &months)
		installment.SetHistorical(isHistorical)
		installments = append(installments, installment)
	}
	return installments
}

func normalizePaidInstallments(msi *int, paidInstallments ...int) (int, error) {
	if len(paidInstallments) == 0 {
		return 0, nil
	}
	paid := paidInstallments[0]
	if paid < 0 {
		return 0, domain.ErrInvalidTransactionMSI
	}
	if msi == nil || *msi <= 1 {
		return 0, nil
	}
	if paid > *msi {
		return 0, domain.ErrInvalidTransactionMSI
	}
	return paid, nil
}

func transactionDeltaForAccount(transactionType domain.TransactionType, accountType domain.FinancialAccountType, amountCents int64) int64 {
	switch transactionType {
	case domain.TransactionTypeIncome:
		return amountCents
	case domain.TransactionTypeExpense:
		if accountType == domain.FinancialAccountTypeCreditCard {
			return amountCents
		}
		return -amountCents
	case domain.TransactionTypeDebtPayment:
		return -amountCents
	default:
		return 0
	}
}

func accountCycleDate(date time.Time, cutoffDay *int, offset int) time.Time {
	if cutoffDay == nil {
		return date.AddDate(0, offset, 0)
	}
	nextDate := billingCycleDate(date.Year(), date.Month()+time.Month(offset), *cutoffDay, date.Location())
	return nextDate
}

func (service *transactionService) GetTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error) {
	return service.getAuthorizedTransaction(ctx, userID, transactionID)
}

func (service *transactionService) GetUserTransactions(ctx context.Context, userID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, filter)
	if err != nil {
		service.logger.Error("failed to retrieve user transactions", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	paidCycles, err := service.transactionRepository.GetPaidCyclesByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve account payable paid cycles", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	for _, transaction := range transactions {
		if transaction != nil {
			transaction.SetPaidCycles(paidCycles[transaction.ID()])
		}
	}

	return transactions, nil
}

func (service *transactionService) UpdateTransaction(
	ctx context.Context,
	userID,
	transactionID string,
	transactionType domain.TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status domain.TransactionStatus,
	msi *int,
	creditCardID *string,
	recurrence domain.TransactionRecurrence,
	recurrenceLimit *int,
) error {
	transaction, err := service.getAuthorizedTransaction(ctx, userID, transactionID)
	if err != nil {
		return err
	}

	if err := transaction.Update(transactionType, concept, category, amountCents, date, status, msi, creditCardID, recurrence, recurrenceLimit); err != nil {
		return err
	}

	return service.persistTransactionUpdate(ctx, transaction)
}

func (service *transactionService) PayAccountPayable(ctx context.Context, userID, transactionID string, dueDate *time.Time) error {
	accountPayable, err := service.getAuthorizedTransaction(ctx, userID, transactionID)
	if err != nil {
		return err
	}
	if accountPayable.Type() != domain.TransactionTypeExpense ||
		accountPayable.Status() != domain.TransactionStatusPending {
		return domain.ErrInvalidTransactionStatus
	}
	cycleDueDate := accountPayable.Date()
	if dueDate != nil {
		cycleDueDate = normalizeCycleDate(*dueDate)
		if !isValidPayableCycleDate(accountPayable.Date(), accountPayable.Recurrence(), cycleDueDate) {
			return domain.ErrInvalidTransactionDate
		}
	}

	paidAt := time.Now()
	paymentID := service.idGenerator.Generate()
	payment, err := domain.NewTransaction(
		paymentID,
		userID,
		domain.TransactionTypeExpense,
		accountPayable.Concept(),
		accountPayable.Category(),
		accountPayable.AmountCents(),
		paidAt,
		domain.TransactionStatusCompleted,
		nil,
		accountPayable.CreditCardID(),
		domain.TransactionRecurrenceOnce,
		nil,
	)
	if err != nil {
		return err
	}

	accountPayable.MarkPaidAt(paidAt)
	paidCycles, err := service.transactionRepository.GetPaidCyclesByAccountPayableID(ctx, accountPayable.ID())
	if err != nil {
		service.logger.Error("failed to retrieve account payable paid cycles", "userID", userID, "transactionID", accountPayable.ID(), "error", err)
		return ErrInternalProcessing
	}
	paidCycles = append(paidCycles, domain.PaidCycle{DueDate: cycleDueDate, PaidAt: paidAt})
	advanceAccountPayableToFirstUnpaidCycle(accountPayable, paidCycles)

	cyclePayment := ports.AccountPayableCyclePayment{
		ID:                   service.idGenerator.Generate(),
		AccountPayableID:     accountPayable.ID(),
		UserID:               userID,
		DueDate:              cycleDueDate,
		PaidAt:               paidAt,
		AmountCents:          accountPayable.AmountCents(),
		Concept:              accountPayable.Concept(),
		Category:             accountPayable.Category(),
		CreatedTransactionID: paymentID,
		CreatedAt:            paidAt,
	}
	if err := service.transactionRepository.CreatePaymentCycleAndUpdate(ctx, payment, accountPayable, cyclePayment); err != nil {
		if errors.Is(err, ports.ErrAccountPayableCycleAlreadyPaid) {
			return ports.ErrAccountPayableCycleAlreadyPaid
		}
		service.logger.Error("failed to pay account payable", "userID", userID, "transactionID", accountPayable.ID(), "paymentID", payment.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func normalizeCycleDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func isValidPayableCycleDate(startDate time.Time, recurrence domain.TransactionRecurrence, dueDate time.Time) bool {
	currentDate := normalizeCycleDate(startDate)
	targetDate := normalizeCycleDate(dueDate)
	for index := 0; index < 1200; index++ {
		if currentDate.Equal(targetDate) {
			return true
		}
		if currentDate.After(targetDate) || recurrence == domain.TransactionRecurrenceOnce {
			return false
		}
		currentDate = recurrence.NextDate(currentDate)
	}
	return false
}

func advanceAccountPayableToFirstUnpaidCycle(accountPayable *domain.Transaction, paidCycles []domain.PaidCycle) {
	if accountPayable.Recurrence() == domain.TransactionRecurrenceOnce {
		for _, paidCycle := range paidCycles {
			if normalizeCycleDate(paidCycle.DueDate).Equal(normalizeCycleDate(accountPayable.Date())) {
				accountPayable.Complete()
				return
			}
		}
		return
	}

	paidCycleDates := make(map[string]struct{}, len(paidCycles))
	for _, paidCycle := range paidCycles {
		paidCycleDates[normalizeCycleDate(paidCycle.DueDate).Format("2006-01-02")] = struct{}{}
	}
	for index := 0; index < 1200; index++ {
		key := normalizeCycleDate(accountPayable.Date()).Format("2006-01-02")
		if _, ok := paidCycleDates[key]; !ok {
			return
		}
		accountPayable.AdvanceRecurrence()
		if accountPayable.Status() == domain.TransactionStatusCompleted {
			return
		}
	}
}

func (service *transactionService) DeleteTransaction(ctx context.Context, userID, transactionID string) error {
	transaction, err := service.getAuthorizedTransaction(ctx, userID, transactionID)
	if err != nil {
		return err
	}

	if err := service.transactionRepository.Delete(ctx, transaction.ID()); err != nil {
		service.logger.Error("failed to delete transaction", "userID", userID, "transactionID", transaction.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *transactionService) GetFinancialSummary(ctx context.Context, userID string, startDate, endDate time.Time) (ports.FinancialSummary, error) {
	if startDate.IsZero() || endDate.IsZero() || !endDate.After(startDate) {
		return ports.FinancialSummary{}, domain.ErrInvalidTransactionDate
	}

	filter := ports.TransactionDateFilter{From: &startDate, To: &endDate}
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, filter)
	if err != nil {
		service.logger.Error("failed to retrieve transactions for financial summary", "userID", userID, "error", err)
		return ports.FinancialSummary{}, ErrInternalProcessing
	}

	return calculateFinancialSummary(transactions), nil
}

func (service *transactionService) getAuthorizedTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error) {
	transaction, err := service.transactionRepository.GetByID(ctx, transactionID)
	if errors.Is(err, ports.ErrTransactionNotFound) {
		return nil, ports.ErrTransactionNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve transaction", "userID", userID, "transactionID", transactionID, "error", err)
		return nil, ErrInternalProcessing
	}
	if transaction == nil {
		return nil, ports.ErrTransactionNotFound
	}
	if transaction.UserID() != userID {
		service.logger.Warn("unauthorized transaction access attempt", "userID", userID, "transactionID", transactionID)
		return nil, ports.ErrTransactionNotFound
	}

	return transaction, nil
}

func (service *transactionService) persistTransactionUpdate(ctx context.Context, transaction *domain.Transaction) error {
	if err := service.transactionRepository.Update(ctx, transaction); err != nil {
		service.logger.Error("failed to update transaction", "userID", transaction.UserID(), "transactionID", transaction.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func calculateFinancialSummary(transactions []*domain.Transaction) ports.FinancialSummary {
	var summary ports.FinancialSummary
	for _, transaction := range transactions {
		if transaction == nil ||
			transaction.IsHistorical() ||
			(transaction.Status() != domain.TransactionStatusPaid &&
				transaction.Status() != domain.TransactionStatusCompleted) ||
			isCompletedAccountPayable(transaction) {
			continue
		}

		switch transaction.Type() {
		case domain.TransactionTypeIncome:
			summary.TotalIncomeCents += transaction.AmountCents()
		case domain.TransactionTypeExpense:
			summary.TotalExpenseCents += transaction.AmountCents()
		}
	}

	summary.ProfitMarginCents = summary.TotalIncomeCents - summary.TotalExpenseCents
	return summary
}

func isCompletedAccountPayable(transaction *domain.Transaction) bool {
	recurrenceLimit := transaction.RecurrenceLimit()
	return transaction.Status() == domain.TransactionStatusCompleted &&
		recurrenceLimit != nil &&
		*recurrenceLimit == 0
}
