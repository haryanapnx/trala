package domain

import (
	"context"
	"time"
)

// TaskStatus is a per-user workflow step (e.g. todo, in_progress, done).
type TaskStatus struct {
	ID        string
	UserID    string
	Name      string
	Slug      string
	Position  int32
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TaskStatusRepository is the persistence port for custom task statuses.
type TaskStatusRepository interface {
	Create(ctx context.Context, s *TaskStatus) (*TaskStatus, error)
	ListByUser(ctx context.Context, userID string) ([]TaskStatus, error)
	GetByID(ctx context.Context, userID, id string) (*TaskStatus, error)
	GetBySlug(ctx context.Context, userID, slug string) (*TaskStatus, error)
	GetFirst(ctx context.Context, userID string) (*TaskStatus, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
	MaxPosition(ctx context.Context, userID string) (int32, error)
	Update(ctx context.Context, s *TaskStatus) (*TaskStatus, error)
	SetPosition(ctx context.Context, userID, id string, position int32) error
	Delete(ctx context.Context, userID, id string) error
	IsInUse(ctx context.Context, id string) (bool, error)
}
