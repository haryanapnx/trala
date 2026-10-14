package postgres

import (
	"context"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/repository/postgres/sqlc"
)

type TaskLogRepository struct {
	db *DB
}

func NewTaskLogRepository(db *DB) *TaskLogRepository {
	return &TaskLogRepository{db: db}
}

func (r *TaskLogRepository) Create(ctx context.Context, log *domain.TaskLog) (*domain.TaskLog, error) {
	taskID, err := parseUUID(log.TaskID)
	if err != nil {
		return nil, err
	}
	actorID, err := parseUUID(log.ActorID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).CreateTaskLog(ctx, sqlc.CreateTaskLogParams{
		TaskID:   taskID,
		ActorID:  actorID,
		Action:   log.Action,
		Field:    log.Field,
		OldValue: log.OldValue,
		NewValue: log.NewValue,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &domain.TaskLog{
		ID:        row.ID.String(),
		TaskID:    row.TaskID.String(),
		ActorID:   row.ActorID.String(),
		Action:    row.Action,
		Field:     row.Field,
		OldValue:  row.OldValue,
		NewValue:  row.NewValue,
		CreatedAt: row.CreatedAt,
	}, nil
}

var _ domain.TaskLogRepository = (*TaskLogRepository)(nil)
