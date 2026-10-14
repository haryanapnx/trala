package model

import "github.com/haryfun/task-management-service/internal/domain"

// CreateTaskInput is the input for creating a task.
type CreateTaskInput struct {
	UserID         string
	TeamID         string
	StatusID       string
	Title          string
	Description    *string
	IdempotencyKey string
}

// CreateTaskResult carries the response to write back: either freshly encoded
// or the stored response from the first request with the same idempotency key.
// Task is set only for a fresh creation, and Replayed is true only when the
// response came from the store.
type CreateTaskResult struct {
	Task       *domain.Task
	StatusCode int
	Body       []byte
	Replayed   bool
}

// ListTasksInput is the input for listing tasks with optional filters.
type ListTasksInput struct {
	UserID string
	Status string // status slug; empty means all statuses
	Search string // title search; empty means no search
	Page   int32
	Limit  int32
}

// TaskListResult is a page of tasks plus pagination metadata.
type TaskListResult struct {
	Tasks      []domain.Task
	Page       int32
	Limit      int32
	Total      int64
	TotalPages int32
}

// UpdateTaskInput is the input for a partial task update.
type UpdateTaskInput struct {
	UserID      string
	TaskID      string
	Title       *string
	Description *string
	StatusID    *string
}

// AssignTaskInput is the input for assigning a task.
type AssignTaskInput struct {
	UserID     string
	TaskID     string
	AssigneeID string
}
