package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	creditCardStatementReconciliationInterval = 24 * time.Hour
	creditCardStatementFallbackTimezone       = "America/Mexico_City"
)

func startCreditCardStatementWorker(ctx context.Context, database *sql.DB, useCase ports.CreditCardUseCase, logger ports.Logger) {
	reconcileCreditCardStatements(ctx, database, useCase, logger)
	ticker := time.NewTicker(creditCardStatementReconciliationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcileCreditCardStatements(ctx, database, useCase, logger)
		}
	}
}

func reconcileCreditCardStatements(ctx context.Context, database *sql.DB, useCase ports.CreditCardUseCase, logger ports.Logger) {
	if database == nil || useCase == nil {
		return
	}
	rows, err := database.QueryContext(ctx, `SELECT DISTINCT user_id FROM credit_cards WHERE deleted_at IS NULL`)
	if err != nil {
		logger.Warn("[FINANCE] No se pudieron listar usuarios para conciliar estados de cuenta", "error", err)
		return
	}
	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			logger.Warn("[FINANCE] No se pudo leer usuario para conciliar estados de cuenta", "error", err)
			continue
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		logger.Warn("[FINANCE] La lista de usuarios para conciliación terminó con error", "error", err)
	}
	rows.Close()

	for _, userID := range userIDs {
		if _, err := useCase.GetPayables(ctx, userID, creditCardStatementFallbackTimezone); err != nil {
			logger.Warn("[FINANCE] Conciliación diaria de estados de cuenta falló", "userID", userID, "error", err)
		}
	}
}
