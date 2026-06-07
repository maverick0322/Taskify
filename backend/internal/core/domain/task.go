package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	minTaskTitleLength = 3
)

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

func (status TaskStatus) IsValid() bool {
	return status == TaskStatusTodo ||
		status == TaskStatusInProgress ||
		status == TaskStatusDone
}

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

func (priority TaskPriority) IsValid() bool {
	return priority == TaskPriorityLow ||
		priority == TaskPriorityMedium ||
		priority == TaskPriorityHigh
}

var (
	ErrEmptyTaskID         = errors.New("domain: task ID cannot be empty")
	ErrEmptyTaskUserID     = errors.New("domain: task user ID cannot be empty")
	ErrInvalidTaskTitle    = errors.New("domain: task title does not meet minimum length")
	ErrInvalidTaskStatus   = errors.New("domain: invalid task status")
	ErrInvalidTaskPriority = errors.New("domain: invalid task priority")
	ErrPastDueDate         = errors.New("domain: task due date cannot be in the past")
)

// Task is the aggregate root for personal work tracking.
type Task struct {
	id          string
	userID      string
	title       string
	description string
	status      TaskStatus
	priority    TaskPriority
	createdAt   time.Time
	updatedAt   time.Time
	dueDate     time.Time
}

// NewTask centralizes invariants so invalid task state cannot enter the domain.
func NewTask(id, userID, title, description string, status TaskStatus, priority TaskPriority, dueDate time.Time) (*Task, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrEmptyTaskID
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, ErrEmptyTaskUserID
	}

	trimmedTitle, err := validateTaskTitle(title)
	if err != nil {
		return nil, err
	}

	if !status.IsValid() {
		return nil, ErrInvalidTaskStatus
	}

	if !priority.IsValid() {
		return nil, ErrInvalidTaskPriority
	}

	if isPastDueDate(dueDate, time.Now()) {
		return nil, ErrPastDueDate
	}

	currentTime := time.Now()
	return &Task{
		id:          trimmedID,
		userID:      trimmedUserID,
		title:       trimmedTitle,
		description: strings.TrimSpace(description),
		status:      status,
		priority:    priority,
		createdAt:   currentTime,
		updatedAt:   currentTime,
		dueDate:     dueDate,
	}, nil
}

func (task *Task) ChangeStatus(newStatus TaskStatus) error {
	if !newStatus.IsValid() {
		return ErrInvalidTaskStatus
	}

	task.status = newStatus
	task.touch()
	return nil
}

func (task *Task) UpdateDetails(newTitle, newDescription string) error {
	trimmedTitle, err := validateTaskTitle(newTitle)
	if err != nil {
		return err
	}

	task.title = trimmedTitle
	task.description = strings.TrimSpace(newDescription)
	task.touch()
	return nil
}

func (task *Task) ChangePriority(newPriority TaskPriority) error {
	if !newPriority.IsValid() {
		return ErrInvalidTaskPriority
	}

	task.priority = newPriority
	task.touch()
	return nil
}

func (task *Task) ID() string {
	return task.id
}

func (task *Task) UserID() string {
	return task.userID
}

func (task *Task) Title() string {
	return task.title
}

func (task *Task) Description() string {
	return task.description
}

func (task *Task) Status() TaskStatus {
	return task.status
}

func (task *Task) Priority() TaskPriority {
	return task.priority
}

func (task *Task) CreatedAt() time.Time {
	return task.createdAt
}

func (task *Task) UpdatedAt() time.Time {
	return task.updatedAt
}

func (task *Task) DueDate() time.Time {
	return task.dueDate
}

func (task *Task) touch() {
	task.updatedAt = time.Now()
}

func validateTaskTitle(title string) (string, error) {
	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) < minTaskTitleLength {
		return "", ErrInvalidTaskTitle
	}

	return trimmedTitle, nil
}

func isPastDueDate(dueDate, now time.Time) bool {
	return !dueDate.IsZero() && dueDate.Before(now)
}
