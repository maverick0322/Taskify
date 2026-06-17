package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidNotificationID      = errors.New("domain: notification ID cannot be empty")
	ErrInvalidNotificationUserID  = errors.New("domain: notification user ID cannot be empty")
	ErrInvalidNotificationTitle   = errors.New("domain: notification title cannot be empty")
	ErrInvalidNotificationMessage = errors.New("domain: notification message cannot be empty")
)

type Notification struct {
	id        string
	userID    string
	title     string
	message   string
	isRead    bool
	createdAt time.Time
	updatedAt time.Time
}

func NewNotification(id, userID, title, message string) (*Notification, error) {
	fields, err := validateNotificationFields(id, userID, title, message)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &Notification{
		id:        fields.id,
		userID:    fields.userID,
		title:     fields.title,
		message:   fields.message,
		isRead:    false,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RehydrateNotification(id, userID, title, message string, isRead bool, createdAt, updatedAt time.Time) (*Notification, error) {
	notification, err := NewNotification(id, userID, title, message)
	if err != nil {
		return nil, err
	}
	notification.isRead = isRead
	notification.createdAt = createdAt
	notification.updatedAt = updatedAt
	return notification, nil
}

func (notification *Notification) MarkAsRead() {
	notification.isRead = true
	notification.updatedAt = time.Now()
}

func (notification *Notification) ID() string           { return notification.id }
func (notification *Notification) UserID() string       { return notification.userID }
func (notification *Notification) Title() string        { return notification.title }
func (notification *Notification) Message() string      { return notification.message }
func (notification *Notification) IsRead() bool         { return notification.isRead }
func (notification *Notification) CreatedAt() time.Time { return notification.createdAt }
func (notification *Notification) UpdatedAt() time.Time { return notification.updatedAt }

func validateNotificationFields(id, userID, title, message string) (struct{ id, userID, title, message string }, error) {
	fields := struct{ id, userID, title, message string }{
		id:      strings.TrimSpace(id),
		userID:  strings.TrimSpace(userID),
		title:   strings.TrimSpace(title),
		message: strings.TrimSpace(message),
	}
	if fields.id == "" {
		return fields, ErrInvalidNotificationID
	}
	if fields.userID == "" {
		return fields, ErrInvalidNotificationUserID
	}
	if fields.title == "" {
		return fields, ErrInvalidNotificationTitle
	}
	if fields.message == "" {
		return fields, ErrInvalidNotificationMessage
	}
	return fields, nil
}
