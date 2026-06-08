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
	ErrEmptyTaskID          = errors.New("domain: task ID cannot be empty")
	ErrEmptyTaskUserID      = errors.New("domain: task user ID cannot be empty")
	ErrEmptyTaskBoardID     = errors.New("domain: task board ID cannot be empty")
	ErrInvalidTaskTitle     = errors.New("domain: task title does not meet minimum length")
	ErrInvalidTaskStatus    = errors.New("domain: invalid task status")
	ErrInvalidTaskPriority  = errors.New("domain: invalid task priority")
	ErrPastDueDate          = errors.New("domain: task due date cannot be in the past")
	ErrInvalidTaskCreatedAt = errors.New("domain: task created at cannot be zero")
	ErrInvalidTaskUpdatedAt = errors.New("domain: task updated at cannot be zero")
)

// Task is the aggregate root for personal work tracking.
type Task struct {
	id          string
	userID      string
	boardID     string
	title       string
	description string
	status      TaskStatus
	priority    TaskPriority
	createdAt   time.Time
	updatedAt   time.Time
	dueDate     time.Time
}

// NewTask centralizes invariants so invalid task state cannot enter the domain.
func NewTask(id, userID, boardID, title, description string, status TaskStatus, priority TaskPriority, dueDate time.Time) (*Task, error) {
	taskFields, err := validateTaskFields(id, userID, boardID, title, description, status, priority, dueDate)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	return &Task{
		id:          taskFields.id,
		userID:      taskFields.userID,
		boardID:     taskFields.boardID,
		title:       taskFields.title,
		description: taskFields.description,
		status:      status,
		priority:    priority,
		createdAt:   currentTime,
		updatedAt:   currentTime,
		dueDate:     dueDate,
	}, nil
}

// RehydrateTask restores persisted state without exposing mutation-oriented setters.
func RehydrateTask(
	id,
	userID,
	boardID,
	title,
	description string,
	status TaskStatus,
	priority TaskPriority,
	createdAt,
	updatedAt,
	dueDate time.Time,
) (*Task, error) {
	taskFields, err := validateTaskFields(id, userID, boardID, title, description, status, priority, dueDate)
	if err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidTaskCreatedAt
	}
	if updatedAt.IsZero() {
		return nil, ErrInvalidTaskUpdatedAt
	}

	return &Task{
		id:          taskFields.id,
		userID:      taskFields.userID,
		boardID:     taskFields.boardID,
		title:       taskFields.title,
		description: taskFields.description,
		status:      status,
		priority:    priority,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		dueDate:     dueDate,
	}, nil
}

func validateTaskFields(id, userID, boardID, title, description string, status TaskStatus, priority TaskPriority, dueDate time.Time) (validatedTaskFields, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return validatedTaskFields{}, ErrEmptyTaskID
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return validatedTaskFields{}, ErrEmptyTaskUserID
	}

	trimmedBoardID := strings.TrimSpace(boardID)
	if trimmedBoardID == "" {
		return validatedTaskFields{}, ErrEmptyTaskBoardID
	}

	trimmedTitle, err := validateTaskTitle(title)
	if err != nil {
		return validatedTaskFields{}, err
	}

	if !status.IsValid() {
		return validatedTaskFields{}, ErrInvalidTaskStatus
	}

	if !priority.IsValid() {
		return validatedTaskFields{}, ErrInvalidTaskPriority
	}

	if isPastDueDate(dueDate, time.Now()) {
		return validatedTaskFields{}, ErrPastDueDate
	}

	return validatedTaskFields{
		id:          trimmedID,
		userID:      trimmedUserID,
		boardID:     trimmedBoardID,
		title:       trimmedTitle,
		description: strings.TrimSpace(description),
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

func (task *Task) BoardID() string {
	return task.boardID
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

type validatedTaskFields struct {
	id          string
	userID      string
	boardID     string
	title       string
	description string
}
