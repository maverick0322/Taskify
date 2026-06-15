package services

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const defaultCashAccountName = "Efectivo"

type financialAccountService struct {
	repository            ports.FinancialAccountRepository
	transactionRepository ports.TransactionRepository
	idGenerator           ports.IDGenerator
	logger                ports.Logger
}

func NewFinancialAccountService(repository ports.FinancialAccountRepository, idGenerator ports.IDGenerator, logger ports.Logger, transactionRepository ...ports.TransactionRepository) ports.FinancialAccountUseCase {
	var transactions ports.TransactionRepository
	if len(transactionRepository) > 0 {
		transactions = transactionRepository[0]
	}
	return &financialAccountService{repository: repository, transactionRepository: transactions, idGenerator: idGenerator, logger: logger}
}

func (service *financialAccountService) CreateAccount(ctx context.Context, userID string, accountType domain.FinancialAccountType, name, institution string, last4 *string, openingBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int, color string) (*domain.FinancialAccount, error) {
	accountID := service.idGenerator.Generate()
	account, err := domain.NewFinancialAccount(accountID, userID, accountType, name, institution, last4, openingBalanceCents, openingBalanceCents, creditLimitCents, cutoffDay, paymentDay, color)
	if err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, account); err != nil {
		service.logger.Error("failed to create financial account", "userID", userID, "accountID", accountID, "error", err)
		return nil, ErrInternalProcessing
	}
	return account, nil
}

func (service *financialAccountService) GetAccounts(ctx context.Context, userID string) ([]*domain.FinancialAccount, error) {
	accounts, err := service.repository.GetByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve financial accounts", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	for _, account := range accounts {
		if account != nil && account.Type() == domain.FinancialAccountTypeCash && account.Name() == defaultCashAccountName {
			return accounts, nil
		}
	}
	cashAccount, err := domain.NewFinancialAccount(service.idGenerator.Generate(), userID, domain.FinancialAccountTypeCash, defaultCashAccountName, "", nil, 0, 0, nil, nil, nil, "from-zinc-700 to-zinc-950")
	if err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, cashAccount); err != nil && !errors.Is(err, ports.ErrFinancialAccountAlreadyExists) {
		service.logger.Error("failed to ensure default cash account", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	return append([]*domain.FinancialAccount{cashAccount}, accounts...), nil
}

func (service *financialAccountService) GetAccountSummary(ctx context.Context, userID, accountID string) (ports.FinancialAccountSummary, error) {
	account, err := service.getAuthorizedAccount(ctx, userID, accountID)
	if err != nil {
		return ports.FinancialAccountSummary{}, err
	}

	currentDebtCents := int64(0)
	availableCreditCents := int64(0)
	if account.Type() == domain.FinancialAccountTypeCreditCard {
		currentDebtCents = account.CurrentBalanceCents()
		if account.CreditLimitCents() != nil {
			availableCreditCents = *account.CreditLimitCents() - currentDebtCents
			if availableCreditCents < 0 {
				availableCreditCents = 0
			}
		}
	}

	return ports.FinancialAccountSummary{
		AccountID:            account.ID(),
		CurrentBalanceCents:  account.CurrentBalanceCents(),
		OpeningBalanceCents:  account.OpeningBalanceCents(),
		CreditLimitCents:     account.CreditLimitCents(),
		CurrentDebtCents:     currentDebtCents,
		AvailableCreditCents: availableCreditCents,
		CalculatedAt:         time.Now(),
	}, nil
}

func (service *financialAccountService) GetAccountTransactions(ctx context.Context, userID, accountID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	account, err := service.getAuthorizedAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	if service.transactionRepository == nil {
		service.logger.Error("financial account transactions requested without transaction repository", "userID", userID, "accountID", accountID)
		return nil, ErrInternalProcessing
	}

	transactions, err := service.transactionRepository.GetByPaymentAccountID(ctx, userID, account.ID(), filter)
	if err != nil {
		service.logger.Error("failed to retrieve financial account transactions", "userID", userID, "accountID", account.ID(), "error", err)
		return nil, ErrInternalProcessing
	}
	return transactions, nil
}

func (service *financialAccountService) UpdateAccount(ctx context.Context, userID, accountID string, accountType domain.FinancialAccountType, name, institution string, last4 *string, openingBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int, color string) error {
	existingAccount, err := service.getAuthorizedAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}
	currentBalanceCents := existingAccount.CurrentBalanceCents()
	if existingAccount.OpeningBalanceCents() == existingAccount.CurrentBalanceCents() {
		currentBalanceCents = openingBalanceCents
	}
	account, err := domain.NewFinancialAccount(accountID, userID, accountType, name, institution, last4, openingBalanceCents, currentBalanceCents, creditLimitCents, cutoffDay, paymentDay, color)
	if err != nil {
		return err
	}
	if err := service.repository.Update(ctx, account); err != nil {
		service.logger.Error("failed to update financial account", "userID", userID, "accountID", accountID, "error", err)
		return ErrInternalProcessing
	}
	return nil
}

func (service *financialAccountService) DeleteAccount(ctx context.Context, userID, accountID string) error {
	account, err := service.getAuthorizedAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, account.ID()); err != nil {
		service.logger.Error("failed to delete financial account", "userID", userID, "accountID", accountID, "error", err)
		return ErrInternalProcessing
	}
	return nil
}

func (service *financialAccountService) getAuthorizedAccount(ctx context.Context, userID, accountID string) (*domain.FinancialAccount, error) {
	account, err := service.repository.GetByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, ports.ErrFinancialAccountNotFound) {
			return nil, ports.ErrFinancialAccountNotFound
		}
		service.logger.Error("failed to retrieve financial account", "userID", userID, "accountID", accountID, "error", err)
		return nil, ErrInternalProcessing
	}
	if account.UserID() != userID {
		service.logger.Warn("unauthorized financial account access attempt", "userID", userID, "accountID", accountID)
		return nil, ports.ErrFinancialAccountNotFound
	}
	return account, nil
}
