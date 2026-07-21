package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const defaultFinancialTimezone = "America/Mexico_City"

var creditCardCurrentTime = time.Now

type creditCardService struct {
	creditCardRepository  ports.CreditCardRepository
	transactionRepository ports.TransactionRepository
	accountRepository     ports.FinancialAccountRepository
	statementRepository   ports.CreditCardStatementRepository
	idGenerator           ports.IDGenerator
	logger                ports.Logger
}

func NewCreditCardService(creditCardRepository ports.CreditCardRepository, transactionRepository ports.TransactionRepository, idGenerator ports.IDGenerator, logger ports.Logger, accountRepository ...ports.FinancialAccountRepository) ports.CreditCardUseCase {
	var accounts ports.FinancialAccountRepository
	if len(accountRepository) > 0 {
		accounts = accountRepository[0]
	}
	return &creditCardService{creditCardRepository: creditCardRepository, transactionRepository: transactionRepository, accountRepository: accounts, idGenerator: idGenerator, logger: logger}
}

func NewCreditCardServiceWithStatements(creditCardRepository ports.CreditCardRepository, transactionRepository ports.TransactionRepository, accountRepository ports.FinancialAccountRepository, statementRepository ports.CreditCardStatementRepository, idGenerator ports.IDGenerator, logger ports.Logger) ports.CreditCardUseCase {
	return &creditCardService{creditCardRepository: creditCardRepository, transactionRepository: transactionRepository, accountRepository: accountRepository, statementRepository: statementRepository, idGenerator: idGenerator, logger: logger}
}

func (service *creditCardService) CreateCreditCard(ctx context.Context, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color, network string) (*domain.CreditCard, error) {
	creditCardID := service.idGenerator.Generate()
	creditCard, err := domain.NewCreditCard(creditCardID, userID, name, bank, last4, cutoffDay, paymentDay, limitCents, color, network)
	if err != nil {
		return nil, err
	}
	if err := service.creditCardRepository.Create(ctx, creditCard); err != nil {
		service.logger.Error("failed to create credit card", "userID", userID, "creditCardID", creditCardID, "error", err)
		return nil, ErrInternalProcessing
	}
	return creditCard, nil
}

func (service *creditCardService) GetCardsWithSummary(ctx context.Context, userID string) ([]ports.CreditCardWithSummary, error) {
	return service.getCardsWithSummary(ctx, userID, defaultFinancialTimezone)
}

func (service *creditCardService) getCardsWithSummary(ctx context.Context, userID, timezone string) ([]ports.CreditCardWithSummary, error) {
	cards, err := service.creditCardRepository.GetByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve credit cards", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, ports.TransactionDateFilter{})
	if err != nil {
		service.logger.Error("failed to retrieve transactions for credit card statements", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	rawNow := creditCardCurrentTime()
	now := rawNow.In(financialLocation(timezone))
	summaries := make([]ports.CreditCardWithSummary, 0, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}
		if service.statementRepository == nil {
			startDate, endDate := currentBillingCycle(card.CutoffDay(), rawNow)
			filtered, filterErr := service.transactionRepository.GetByCreditCardID(ctx, userID, card.ID(), ports.TransactionDateFilter{From: &startDate, To: &endDate})
			if filterErr != nil {
				return nil, ErrInternalProcessing
			}
			summaries = append(summaries, ports.CreditCardWithSummary{CreditCard: card, CurrentDebtCents: calculateCreditCardDebt(filtered)})
			continue
		}

		statements, items, allocations := projectCreditCardStatements(card, transactions, now)
		if err := service.statementRepository.Reconcile(ctx, userID, card.ID(), statements, items, allocations); err != nil {
			service.logger.Error("failed to reconcile credit card statements", "userID", userID, "creditCardID", card.ID(), "error", err)
			return nil, ErrInternalProcessing
		}
		debt, dueDate, status := summarizeOutstandingStatements(statements, now)
		totalBalance := int64(0)
		if service.accountRepository != nil {
			if account, accountErr := service.accountRepository.GetByID(ctx, card.ID()); accountErr == nil && account != nil {
				totalBalance = account.CurrentBalanceCents()
			}
		}
		summaries = append(summaries, ports.CreditCardWithSummary{CreditCard: card, CurrentDebtCents: debt, TotalBalanceCents: totalBalance, PaymentDueDate: dueDate, Status: status})
	}
	return summaries, nil
}

func (service *creditCardService) GetPayables(ctx context.Context, userID, timezone string) ([]ports.FinancialPayable, error) {
	location := financialLocation(timezone)
	now := creditCardCurrentTime().In(location)
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, ports.TransactionDateFilter{})
	if err != nil {
		service.logger.Error("failed to retrieve global account payables", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	paidCycles, err := service.transactionRepository.GetPaidCyclesByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve paid account payable cycles", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	payables := manualFinancialPayables(transactions, paidCycles, now)
	summaries, err := service.getCardsWithSummary(ctx, userID, timezone)
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		if summary.CreditCard == nil || summary.CurrentDebtCents <= 0 || summary.PaymentDueDate == nil {
			continue
		}
		cardID := summary.CreditCard.ID()
		payables = append(payables, ports.FinancialPayable{ID: "credit-card-" + cardID, Type: "credit_card", SourceID: cardID, Name: summary.CreditCard.Name(), AmountCents: summary.CurrentDebtCents, DueDate: *summary.PaymentDueDate, Status: summary.Status, CreditCardID: &cardID})
	}
	sort.Slice(payables, func(i, j int) bool { return payables[i].DueDate.Before(payables[j].DueDate) })
	return payables, nil
}

func (service *creditCardService) UpdateCreditCard(ctx context.Context, userID, creditCardID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color, network string) error {
	creditCard, err := service.getAuthorizedCreditCard(ctx, userID, creditCardID)
	if err != nil {
		return err
	}
	if err := creditCard.Update(name, bank, last4, cutoffDay, paymentDay, limitCents, color, network); err != nil {
		return err
	}
	if err := service.creditCardRepository.Update(ctx, creditCard); err != nil {
		service.logger.Error("failed to update credit card", "userID", userID, "creditCardID", creditCard.ID(), "error", err)
		return ErrInternalProcessing
	}
	return nil
}

func (service *creditCardService) PayCreditCardDebt(ctx context.Context, userID, creditCardID, sourceAccountID string, amountCents int64, timezone ...string) (ports.CreditCardPaymentResult, error) {
	result := ports.CreditCardPaymentResult{}
	if amountCents <= 0 {
		return result, domain.ErrInvalidTransactionAmount
	}
	card, err := service.getAuthorizedCreditCard(ctx, userID, creditCardID)
	if err != nil {
		return result, err
	}
	if service.accountRepository == nil {
		return result, ErrInternalProcessing
	}
	sourceAccount, err := service.accountRepository.GetByID(ctx, sourceAccountID)
	if errors.Is(err, ports.ErrFinancialAccountNotFound) || sourceAccount == nil || sourceAccount.UserID() != userID || sourceAccount.Type() == domain.FinancialAccountTypeCreditCard {
		return result, ports.ErrFinancialAccountNotFound
	}
	if err != nil {
		return result, ErrInternalProcessing
	}
	creditAccount, err := service.accountRepository.GetByID(ctx, creditCardID)
	if errors.Is(err, ports.ErrFinancialAccountNotFound) || creditAccount == nil || creditAccount.UserID() != userID || creditAccount.Type() != domain.FinancialAccountTypeCreditCard {
		return result, ports.ErrCreditCardNotFound
	}
	if err != nil {
		return result, ErrInternalProcessing
	}
	if err := sourceAccount.ApplyDelta(-amountCents); err != nil {
		return result, err
	}

	zone := defaultFinancialTimezone
	if len(timezone) > 0 && strings.TrimSpace(timezone[0]) != "" {
		zone = timezone[0]
	}
	now := creditCardCurrentTime().In(financialLocation(zone))
	if service.statementRepository == nil {
		return service.payCreditCardWithoutStatements(ctx, card, sourceAccountID, creditAccount, amountCents, now)
	}
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, ports.TransactionDateFilter{})
	if err != nil {
		return result, ErrInternalProcessing
	}
	statements, items, historicalAllocations := projectCreditCardStatements(card, transactions, now)
	if err := service.statementRepository.Reconcile(ctx, userID, card.ID(), statements, items, historicalAllocations); err != nil {
		service.logger.Error("failed to reconcile statements before card payment", "userID", userID, "creditCardID", card.ID(), "error", err)
		return result, ErrInternalProcessing
	}
	outstanding := outstandingStatements(statements)
	totalDebt := statementOutstandingTotal(outstanding)
	if totalDebt <= 0 {
		return result, ports.ErrCreditCardNoPayableDebt
	}
	if amountCents > totalDebt {
		return result, ports.ErrCreditCardPaymentExceedsDebt
	}
	if amountCents > creditAccount.CurrentBalanceCents() {
		service.logger.Error("credit card statement debt exceeds account balance", "userID", userID, "creditCardID", creditCardID, "payableDebtCents", totalDebt, "totalBalanceCents", creditAccount.CurrentBalanceCents())
		return result, ports.ErrCreditCardStatementUnavailable
	}

	paymentID := service.idGenerator.Generate()
	payment, err := domain.NewTransaction(paymentID, userID, domain.TransactionTypeDebtPayment, "Pago "+card.Name(), "Tarjeta de credito", amountCents, now, domain.TransactionStatusCompleted, nil, &creditCardID, domain.TransactionRecurrenceOnce, nil)
	if err != nil {
		return result, err
	}
	payment.SetAccountingDetails(&sourceAccountID, &creditCardID, nil, nil)
	payment.SetCreditCardID(&creditCardID)
	allocations, updatedStatements := allocatePayment(userID, paymentID, amountCents, outstanding, now)
	deltas, entries := service.creditCardPaymentAccounting(userID, sourceAccountID, creditCardID, paymentID, amountCents, now)
	if err := service.statementRepository.ApplyPayment(ctx, payment, deltas, entries, allocations, updatedStatements); err != nil {
		service.logger.Error("failed to persist atomic credit card payment", "userID", userID, "creditCardID", creditCardID, "sourceAccountID", sourceAccountID, "amountCents", amountCents, "error", err)
		if errors.Is(err, ports.ErrCreditCardPaymentConflict) {
			return result, ports.ErrCreditCardPaymentConflict
		}
		return result, ErrInternalProcessing
	}
	result = ports.CreditCardPaymentResult{PaymentTransactionID: paymentID, AppliedAmountCents: amountCents, RemainingDebtCents: totalDebt - amountCents, AffectedStatementIDs: make([]string, 0, len(updatedStatements))}
	for _, statement := range updatedStatements {
		result.AffectedStatementIDs = append(result.AffectedStatementIDs, statement.ID)
	}
	return result, nil
}

func (service *creditCardService) payCreditCardWithoutStatements(ctx context.Context, card *domain.CreditCard, sourceAccountID string, creditAccount *domain.FinancialAccount, amountCents int64, now time.Time) (ports.CreditCardPaymentResult, error) {
	if amountCents > creditAccount.CurrentBalanceCents() {
		return ports.CreditCardPaymentResult{}, domain.ErrInvalidTransactionAmount
	}
	cardID := card.ID()
	paymentID := service.idGenerator.Generate()
	payment, err := domain.NewTransaction(paymentID, card.UserID(), domain.TransactionTypeDebtPayment, "Pago "+card.Name(), "Tarjeta de credito", amountCents, now, domain.TransactionStatusCompleted, nil, &cardID, domain.TransactionRecurrenceOnce, nil)
	if err != nil {
		return ports.CreditCardPaymentResult{}, err
	}
	payment.SetAccountingDetails(&sourceAccountID, &cardID, nil, nil)
	payment.SetCreditCardID(&cardID)
	deltas, entries := service.creditCardPaymentAccounting(card.UserID(), sourceAccountID, cardID, paymentID, amountCents, now)
	if err := service.transactionRepository.CreateManyWithLedger(ctx, []*domain.Transaction{payment}, deltas, entries); err != nil {
		return ports.CreditCardPaymentResult{}, ErrInternalProcessing
	}
	return ports.CreditCardPaymentResult{PaymentTransactionID: paymentID, AppliedAmountCents: amountCents, RemainingDebtCents: maxInt64(creditAccount.CurrentBalanceCents()-amountCents, 0)}, nil
}

func (service *creditCardService) creditCardPaymentAccounting(userID, sourceAccountID, cardID, paymentID string, amountCents int64, now time.Time) ([]ports.AccountBalanceDelta, []ports.LedgerEntry) {
	deltas := []ports.AccountBalanceDelta{{AccountID: sourceAccountID, DeltaCents: -amountCents}, {AccountID: cardID, DeltaCents: -amountCents}}
	entries := []ports.LedgerEntry{
		{ID: service.idGenerator.Generate(), UserID: userID, AccountID: sourceAccountID, TransactionID: paymentID, AmountCents: -amountCents, EntryType: string(domain.TransactionTypeDebtPayment), CreatedAt: now, UpdatedAt: now},
		{ID: service.idGenerator.Generate(), UserID: userID, AccountID: cardID, TransactionID: paymentID, AmountCents: -amountCents, EntryType: string(domain.TransactionTypeDebtPayment), CreatedAt: now, UpdatedAt: now},
	}
	return deltas, entries
}

func (service *creditCardService) DeleteCreditCard(ctx context.Context, userID, creditCardID string) error {
	card, err := service.getAuthorizedCreditCard(ctx, userID, creditCardID)
	if err != nil {
		return err
	}
	if err := service.creditCardRepository.Delete(ctx, card.ID()); err != nil {
		return ErrInternalProcessing
	}
	return nil
}

func (service *creditCardService) getAuthorizedCreditCard(ctx context.Context, userID, creditCardID string) (*domain.CreditCard, error) {
	card, err := service.creditCardRepository.GetByID(ctx, creditCardID)
	if errors.Is(err, ports.ErrCreditCardNotFound) || card == nil {
		return nil, ports.ErrCreditCardNotFound
	}
	if err != nil {
		return nil, ErrInternalProcessing
	}
	if card.UserID() != userID {
		return nil, ports.ErrCreditCardNotFound
	}
	return card, nil
}

func projectCreditCardStatements(card *domain.CreditCard, transactions []*domain.Transaction, now time.Time) ([]ports.CreditCardStatement, []ports.CreditCardStatementItem, []ports.CreditCardPaymentAllocation) {
	type projection struct {
		statement      ports.CreditCardStatement
		historicalPaid int64
	}
	byCycle := map[string]*projection{}
	items := make([]ports.CreditCardStatementItem, 0)
	payments := make([]*domain.Transaction, 0)
	for _, transaction := range transactions {
		if transaction == nil || transaction.UserID() != card.UserID() {
			continue
		}
		belongsToCard := transaction.CreditCardID() != nil && *transaction.CreditCardID() == card.ID()
		if transaction.Type() == domain.TransactionTypeDebtPayment {
			fallbackDestination := transaction.DestinationAccountID() != nil && *transaction.DestinationAccountID() == card.ID()
			if (belongsToCard || fallbackDestination) && transactionStatusSettled(transaction.Status()) {
				payments = append(payments, transaction)
			}
			continue
		}
		if !belongsToCard || transaction.Type() != domain.TransactionTypeExpense || !creditCardExpenseBillable(transaction) {
			continue
		}
		cycleStart, cycleEnd := statementCycleForDate(transaction.Date().In(now.Location()), card.CutoffDay())
		if cycleEnd.After(financialDay(now)) {
			continue
		}
		key := cycleEnd.Format("2006-01-02")
		current := byCycle[key]
		if current == nil {
			statementID := deterministicFinancialID("statement", card.ID(), key)
			current = &projection{statement: ports.CreditCardStatement{ID: statementID, UserID: card.UserID(), CreditAccountID: card.ID(), CycleStart: cycleStart, CycleEnd: cycleEnd, PaymentDueDate: paymentDueDate(cycleEnd, card.PaymentDay()), Status: "DUE", CreatedAt: cycleEnd.UTC(), UpdatedAt: cycleEnd.UTC()}}
			byCycle[key] = current
		}
		current.statement.StatementAmountCents += transaction.AmountCents()
		if transaction.IsHistorical() {
			current.historicalPaid += transaction.AmountCents()
		}
		items = append(items, ports.CreditCardStatementItem{ID: deterministicFinancialID("statement-item", current.statement.ID, transaction.ID()), UserID: card.UserID(), StatementID: current.statement.ID, TransactionID: transaction.ID(), AmountCents: transaction.AmountCents(), CreatedAt: cycleEnd.UTC(), UpdatedAt: cycleEnd.UTC()})
	}
	statements := make([]ports.CreditCardStatement, 0, len(byCycle))
	historicalPaid := make(map[string]int64, len(byCycle))
	for _, value := range byCycle {
		value.statement.PaidAmountCents = value.historicalPaid
		statements = append(statements, value.statement)
		historicalPaid[value.statement.ID] = value.historicalPaid
	}
	sort.Slice(statements, func(i, j int) bool { return statements[i].CycleEnd.Before(statements[j].CycleEnd) })
	sort.Slice(payments, func(i, j int) bool { return payments[i].Date().Before(payments[j].Date()) })
	allocations := make([]ports.CreditCardPaymentAllocation, 0)
	for _, payment := range payments {
		remaining := payment.AmountCents()
		for index := range statements {
			statement := &statements[index]
			if remaining <= 0 || statement.CycleEnd.After(payment.Date().In(now.Location())) {
				continue
			}
			outstanding := statement.StatementAmountCents - statement.PaidAmountCents
			if outstanding <= 0 {
				continue
			}
			applied := minInt64(remaining, outstanding)
			statement.PaidAmountCents += applied
			if payment.UpdatedAt().After(statement.UpdatedAt) {
				statement.UpdatedAt = payment.UpdatedAt().UTC()
			}
			remaining -= applied
			allocations = append(allocations, ports.CreditCardPaymentAllocation{ID: deterministicFinancialID("payment-allocation", statement.ID, payment.ID()), UserID: card.UserID(), StatementID: statement.ID, PaymentTransactionID: payment.ID(), AmountCents: applied, CreatedAt: payment.Date().UTC(), UpdatedAt: payment.UpdatedAt().UTC()})
		}
	}
	for index := range statements {
		setStatementStatus(&statements[index], now)
		_ = historicalPaid
	}
	return statements, items, allocations
}

func creditCardExpenseBillable(transaction *domain.Transaction) bool {
	if transaction.InstallmentNumber() != nil {
		return transaction.Status() == domain.TransactionStatusPending || transactionStatusSettled(transaction.Status())
	}
	return transactionStatusSettled(transaction.Status())
}

func transactionStatusSettled(status domain.TransactionStatus) bool {
	return status == domain.TransactionStatusPaid || status == domain.TransactionStatusCompleted
}

func statementCycleForDate(date time.Time, cutoffDay int) (time.Time, time.Time) {
	cycleEnd := billingCycleDate(date.Year(), date.Month(), cutoffDay, date.Location())
	if date.After(cycleEnd.Add(24*time.Hour - time.Nanosecond)) {
		cycleEnd = billingCycleDate(date.Year(), date.Month()+1, cutoffDay, date.Location())
	}
	cycleStart := billingCycleDate(cycleEnd.Year(), cycleEnd.Month()-1, cutoffDay, date.Location())
	return cycleStart, cycleEnd
}

func paymentDueDate(cycleEnd time.Time, paymentDay int) time.Time {
	due := billingCycleDate(cycleEnd.Year(), cycleEnd.Month(), paymentDay, cycleEnd.Location())
	if !due.After(cycleEnd) {
		due = billingCycleDate(cycleEnd.Year(), cycleEnd.Month()+1, paymentDay, cycleEnd.Location())
	}
	return due
}

func summarizeOutstandingStatements(statements []ports.CreditCardStatement, now time.Time) (int64, *time.Time, string) {
	outstanding := outstandingStatements(statements)
	if len(outstanding) == 0 {
		return 0, nil, "paid"
	}
	total := statementOutstandingTotal(outstanding)
	due := outstanding[0].PaymentDueDate
	status := "current"
	for _, statement := range outstanding {
		if statement.PaymentDueDate.Before(financialDay(now)) {
			status = "overdue"
			break
		}
	}
	return total, &due, status
}

func outstandingStatements(statements []ports.CreditCardStatement) []ports.CreditCardStatement {
	result := make([]ports.CreditCardStatement, 0)
	for _, statement := range statements {
		if statement.StatementAmountCents-statement.PaidAmountCents > 0 {
			result = append(result, statement)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PaymentDueDate.Before(result[j].PaymentDueDate) })
	return result
}

func statementOutstandingTotal(statements []ports.CreditCardStatement) int64 {
	var total int64
	for _, statement := range statements {
		total += maxInt64(statement.StatementAmountCents-statement.PaidAmountCents, 0)
	}
	return total
}

func allocatePayment(userID, paymentID string, amount int64, statements []ports.CreditCardStatement, now time.Time) ([]ports.CreditCardPaymentAllocation, []ports.CreditCardStatement) {
	remaining := amount
	allocations := make([]ports.CreditCardPaymentAllocation, 0)
	updated := make([]ports.CreditCardStatement, 0)
	for _, statement := range statements {
		if remaining <= 0 {
			break
		}
		outstanding := statement.StatementAmountCents - statement.PaidAmountCents
		if outstanding <= 0 {
			continue
		}
		applied := minInt64(remaining, outstanding)
		statement.PaidAmountCents += applied
		statement.UpdatedAt = now.UTC()
		setStatementStatus(&statement, now)
		allocations = append(allocations, ports.CreditCardPaymentAllocation{ID: deterministicFinancialID("payment-allocation", statement.ID, paymentID), UserID: userID, StatementID: statement.ID, PaymentTransactionID: paymentID, AmountCents: applied, CreatedAt: now.UTC(), UpdatedAt: now.UTC()})
		updated = append(updated, statement)
		remaining -= applied
	}
	return allocations, updated
}

func setStatementStatus(statement *ports.CreditCardStatement, now time.Time) {
	if statement.PaidAmountCents >= statement.StatementAmountCents {
		statement.PaidAmountCents = statement.StatementAmountCents
		statement.Status = "PAID"
	} else if statement.PaymentDueDate.Before(financialDay(now)) {
		statement.Status = "OVERDUE"
		overdueAt := statement.PaymentDueDate.AddDate(0, 0, 1).UTC()
		if overdueAt.After(statement.UpdatedAt) {
			statement.UpdatedAt = overdueAt
		}
	} else {
		statement.Status = "DUE"
	}
}

func manualFinancialPayables(transactions []*domain.Transaction, paidCycles map[string][]domain.PaidCycle, now time.Time) []ports.FinancialPayable {
	result := make([]ports.FinancialPayable, 0)
	today := financialDay(now)
	for _, transaction := range transactions {
		if transaction == nil || transaction.Type() != domain.TransactionTypeExpense || transaction.Status() != domain.TransactionStatusPending {
			continue
		}
		paid := map[string]struct{}{}
		for _, cycle := range paidCycles[transaction.ID()] {
			paid[cycle.DueDate.In(now.Location()).Format("2006-01-02")] = struct{}{}
		}
		due := financialDay(transaction.Date().In(now.Location()))
		limit := transaction.RecurrenceLimit()
		for index := 0; index < 1200; index++ {
			if limit != nil && index >= *limit {
				break
			}
			key := due.Format("2006-01-02")
			if _, isPaid := paid[key]; !isPaid {
				transactionID := transaction.ID()
				status := "current"
				if due.Before(today) {
					status = "overdue"
				}
				sourceDate := transaction.Date()
				result = append(result, ports.FinancialPayable{ID: deterministicFinancialID("manual-payable", transaction.ID(), key), Type: "manual", SourceID: transaction.ID(), Name: transaction.Concept(), AmountCents: transaction.AmountCents(), DueDate: due, Status: status, TransactionID: &transactionID, Category: transaction.Category(), SourceDate: &sourceDate, Recurrence: string(transaction.Recurrence()), RecurrenceLimit: transaction.RecurrenceLimit()})
			}
			if transaction.Recurrence() == domain.TransactionRecurrenceOnce || !due.Before(today) {
				break
			}
			due = transaction.Recurrence().NextDate(due)
		}
	}
	return result
}

func financialLocation(name string) *time.Location {
	location, err := time.LoadLocation(strings.TrimSpace(name))
	if err == nil {
		return location
	}
	location, err = time.LoadLocation(defaultFinancialTimezone)
	if err == nil {
		return location
	}
	return time.UTC
}

func financialDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func deterministicFinancialID(kind string, parts ...string) string {
	digest := sha256.Sum256([]byte(kind + ":" + strings.Join(parts, ":")))
	bytes := digest[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x50
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func calculateCreditCardDebt(transactions []*domain.Transaction) int64 {
	var debt int64
	for _, transaction := range transactions {
		if transaction == nil || transaction.IsHistorical() || !transactionStatusSettled(transaction.Status()) {
			continue
		}
		switch transaction.Type() {
		case domain.TransactionTypeExpense:
			debt += transaction.AmountCents()
		case domain.TransactionTypeDebtPayment:
			debt -= transaction.AmountCents()
		}
	}
	return maxInt64(debt, 0)
}

func currentBillingCycle(cutoffDay int, now time.Time) (time.Time, time.Time) {
	currentCutoff := billingCycleDate(now.Year(), now.Month(), cutoffDay, now.Location())
	if financialDay(now).After(currentCutoff) {
		return currentCutoff, billingCycleDate(now.Year(), now.Month()+1, cutoffDay, now.Location())
	}
	return billingCycleDate(now.Year(), now.Month()-1, cutoffDay, now.Location()), currentCutoff
}

func billingCycleDate(year int, month time.Month, day int, location *time.Location) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, location)
	lastDay := first.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, location)
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
