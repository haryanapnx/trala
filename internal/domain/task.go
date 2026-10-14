package domain

import (
	"context"
	"time"
)

// Task belongs to a team and is created by a user. Assignee is optional.
type Task struct {
	ID          string
	TeamID      string
	CreatorID   string
	AssigneeID  *string
	StatusID    string
	Title       string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// TaskFilter narrows a task listing. UserID is always applied (creator OR
// assignee); StatusID and Search are optional (empty means "no filter").
type TaskFilter struct {
	UserID   string
	StatusID string
	Search   string
	Limit    int32
	Offset   int32
}

// TaskRepository is the persistence port for tasks.
type TaskRepository interface {
	Create(ctx context.Context, t *Task) (*Task, error)
	GetByID(ctx context.Context, id string) (*Task, error)
	List(ctx context.Context, filter TaskFilter) ([]Task, error)
	Count(ctx context.Context, filter TaskFilter) (int64, error)
	Update(ctx context.Context, t *Task) (*Task, error)
	Assign(ctx context.Context, taskID, creatorID, assigneeID string) (*Task, error)
	SoftDelete(ctx context.Context, id, creatorID string) error
}
