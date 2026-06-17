package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type NotificationHandler struct {
	useCase ports.NotificationUseCase
	logger  ports.Logger
}

func NewNotificationHandler(useCase ports.NotificationUseCase, logger ports.Logger) *NotificationHandler {
	return &NotificationHandler{useCase: useCase, logger: logger}
}

func (handler *NotificationHandler) RegisterRoutes(router chi.Router) {
	router.Get("/notifications", handler.GetNotifications)
	router.Patch("/notifications/{id}/read", handler.MarkNotificationAsRead)
}

func (handler *NotificationHandler) GetNotifications(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	notifications, err := handler.useCase.GetNotifications(request.Context(), userID)
	if err != nil {
		handler.handleError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, notificationResponsesFromDomain(notifications))
}

func (handler *NotificationHandler) MarkNotificationAsRead(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	if err := handler.useCase.MarkNotificationAsRead(request.Context(), userID, chi.URLParam(request, "id")); err != nil {
		handler.handleError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (handler *NotificationHandler) handleError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrNotificationNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "notification not found"})
	case errors.Is(err, ports.ErrNotificationRepositoryUnavailable):
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	default:
		handler.logger.Error("unexpected notification error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

type notificationResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	IsRead    bool   `json:"isRead"`
	CreatedAt string `json:"createdAt"`
}

func notificationResponsesFromDomain(notifications []*domain.Notification) []notificationResponse {
	responses := make([]notificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		responses = append(responses, notificationResponse{
			ID:        notification.ID(),
			UserID:    notification.UserID(),
			Title:     notification.Title(),
			Message:   notification.Message(),
			IsRead:    notification.IsRead(),
			CreatedAt: notification.CreatedAt().UTC().Format(time.RFC3339),
		})
	}
	return responses
}
