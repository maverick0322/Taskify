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
	ErrInvalidTaskTitle     = errors.New("domain: task title does not meet minimum length")
	ErrInvalidTaskStatus    = errors.New("domain: invalid task status")
	ErrInvalidTaskPriority  = errors.New("domain: invalid task priority")
	ErrInvalidTaskCreatedAt = errors.New("domain: task created at cannot be zero")
	ErrInvalidTaskUpdatedAt = errors.New("domain: task updated at cannot be zero")
)

// Task is the aggregate root for personal work tracking.
type Task struct {
	id          string
	userID      string
	boardID     *string
	columnID    *string
	title       string
	description string
	status      TaskStatus
	priority    TaskPriority
	createdAt   time.Time
	updatedAt   time.Time
	dueDate     time.Time
}

// NewTask centralizes invariants so invalid task state cannot enter the domain.
func NewTask(id, userID string, boardID *string, taskOptions ...interface{}) (*Task, error) {
	columnID, title, description, status, priority, dueDate, err := parseTaskOptions(taskOptions...)
	if err != nil {
		return nil, err
	}
	taskFields, err := validateTaskFields(id, userID, boardID, columnID, title, description, status, priority, dueDate)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	return &Task{
		id:          taskFields.id,
		userID:      taskFields.userID,
		boardID:     taskFields.boardID,
		columnID:    taskFields.columnID,
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
	userID string,
	boardID *string,
	taskOptions ...interface{},
) (*Task, error) {
	columnID, title, description, status, priority, createdAt, updatedAt, dueDate, err := parseRehydrateTaskOptions(taskOptions...)
	if err != nil {
		return nil, err
	}
	taskFields, err := validateTaskFields(id, userID, boardID, columnID, title, description, status, priority, dueDate)
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
		columnID:    taskFields.columnID,
		title:       taskFields.title,
		description: taskFields.description,
		status:      status,
		priority:    priority,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		dueDate:     dueDate,
	}, nil
}

func validateTaskFields(id, userID string, boardID, columnID *string, title, description string, status TaskStatus, priority TaskPriority, dueDate time.Time) (validatedTaskFields, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return validatedTaskFields{}, ErrEmptyTaskID
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return validatedTaskFields{}, ErrEmptyTaskUserID
	}

	trimmedBoardID := normalizeOptionalTaskBoardID(boardID)
	trimmedColumnID := normalizeOptionalTaskBoardID(columnID)

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

	return validatedTaskFields{
		id:          trimmedID,
		userID:      trimmedUserID,
		boardID:     trimmedBoardID,
		columnID:    trimmedColumnID,
		title:       trimmedTitle,
		description: strings.TrimSpace(description),
	}, nil
}

func (task *Task) MoveToColumn(columnID *string) {
	task.columnID = normalizeOptionalTaskBoardID(columnID)
	task.touch()
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

func (task *Task) Update(newTitle, newDescription string, newPriority TaskPriority, newDueDate time.Time) error {
	trimmedTitle, err := validateTaskTitle(newTitle)
	if err != nil {
		return err
	}
	if !newPriority.IsValid() {
		return ErrInvalidTaskPriority
	}

	task.title = trimmedTitle
	task.description = strings.TrimSpace(newDescription)
	task.priority = newPriority
	task.dueDate = newDueDate
	task.touch()
	return nil
}

func (task *Task) ID() string {
	return task.id
}

func (task *Task) UserID() string {
	return task.userID
}

func (task *Task) BoardID() *string {
	return task.boardID
}

func (task *Task) ColumnID() *string {
	return task.columnID
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

type validatedTaskFields struct {
	id          string
	userID      string
	boardID     *string
	columnID    *string
	title       string
	description string
}

func normalizeOptionalTaskBoardID(boardID *string) *string {
	if boardID == nil {
		return nil
	}

	trimmedBoardID := strings.TrimSpace(*boardID)
	if trimmedBoardID == "" {
		return nil
	}

	return &trimmedBoardID
}

func parseTaskOptions(taskOptions ...interface{}) (*string, string, string, TaskStatus, TaskPriority, time.Time, error) {
	switch len(taskOptions) {
	case 5:
		title, titleOK := taskOptions[0].(string)
		description, descriptionOK := taskOptions[1].(string)
		status, statusOK := taskOptions[2].(TaskStatus)
		priority, priorityOK := taskOptions[3].(TaskPriority)
		dueDate, dueDateOK := taskOptions[4].(time.Time)
		if !titleOK || !descriptionOK || !statusOK || !priorityOK || !dueDateOK {
			return nil, "", "", "", "", time.Time{}, ErrInvalidTaskStatus
		}
		return nil, title, description, status, priority, dueDate, nil
	case 6:
		columnID, columnOK := taskOptions[0].(*string)
		title, titleOK := taskOptions[1].(string)
		description, descriptionOK := taskOptions[2].(string)
		status, statusOK := taskOptions[3].(TaskStatus)
		priority, priorityOK := taskOptions[4].(TaskPriority)
		dueDate, dueDateOK := taskOptions[5].(time.Time)
		if !columnOK || !titleOK || !descriptionOK || !statusOK || !priorityOK || !dueDateOK {
			return nil, "", "", "", "", time.Time{}, ErrInvalidTaskStatus
		}
		return columnID, title, description, status, priority, dueDate, nil
	default:
		return nil, "", "", "", "", time.Time{}, ErrInvalidTaskStatus
	}
}

func parseRehydrateTaskOptions(taskOptions ...interface{}) (*string, string, string, TaskStatus, TaskPriority, time.Time, time.Time, time.Time, error) {
	switch len(taskOptions) {
	case 7:
		title, titleOK := taskOptions[0].(string)
		description, descriptionOK := taskOptions[1].(string)
		status, statusOK := taskOptions[2].(TaskStatus)
		priority, priorityOK := taskOptions[3].(TaskPriority)
		createdAt, createdOK := taskOptions[4].(time.Time)
		updatedAt, updatedOK := taskOptions[5].(time.Time)
		dueDate, dueDateOK := taskOptions[6].(time.Time)
		if !titleOK || !descriptionOK || !statusOK || !priorityOK || !createdOK || !updatedOK || !dueDateOK {
			return nil, "", "", "", "", time.Time{}, time.Time{}, time.Time{}, ErrInvalidTaskStatus
		}
		return nil, title, description, status, priority, createdAt, updatedAt, dueDate, nil
	case 8:
		columnID, columnOK := taskOptions[0].(*string)
		title, titleOK := taskOptions[1].(string)
		description, descriptionOK := taskOptions[2].(string)
		status, statusOK := taskOptions[3].(TaskStatus)
		priority, priorityOK := taskOptions[4].(TaskPriority)
		createdAt, createdOK := taskOptions[5].(time.Time)
		updatedAt, updatedOK := taskOptions[6].(time.Time)
		dueDate, dueDateOK := taskOptions[7].(time.Time)
		if !columnOK || !titleOK || !descriptionOK || !statusOK || !priorityOK || !createdOK || !updatedOK || !dueDateOK {
			return nil, "", "", "", "", time.Time{}, time.Time{}, time.Time{}, ErrInvalidTaskStatus
		}
		return columnID, title, description, status, priority, createdAt, updatedAt, dueDate, nil
	default:
		return nil, "", "", "", "", time.Time{}, time.Time{}, time.Time{}, ErrInvalidTaskStatus
	}
}
