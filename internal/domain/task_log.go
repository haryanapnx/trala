package domain

import (
	"context"
	"time"
)

// TaskLog is an audit entry recording a change made to a task.
type TaskLog struct {
	ID        string
	TaskID    string
	ActorID   string
	Action    string
	Field     *string
	OldValue  *string
	NewValue  *string
	CreatedAt time.Time
}

// TaskLogRepository is the persistence port for task audit logs.
type TaskLogRepository interface {
	Create(ctx context.Context, log *TaskLog) (*TaskLog, error)
}
