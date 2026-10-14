package postgres

import (
	"context"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/repository/postgres/sqlc"
)

type TaskStatusRepository struct {
	db *DB
}

func NewTaskStatusRepository(db *DB) *TaskStatusRepository {
	return &TaskStatusRepository{db: db}
}

func (r *TaskStatusRepository) Create(ctx context.Context, s *domain.TaskStatus) (*domain.TaskStatus, error) {
	userID, err := parseUUID(s.UserID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).CreateTaskStatus(ctx, sqlc.CreateTaskStatusParams{
		UserID:    userID,
		Name:      s.Name,
		Slug:      s.Slug,
		Position:  s.Position,
		IsDefault: s.IsDefault,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTaskStatus(row), nil
}

func (r *TaskStatusRepository) ListByUser(ctx context.Context, userID string) ([]domain.TaskStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Queries(ctx).ListTaskStatuses(ctx, uid)
	if err != nil {
		return nil, mapError(err)
	}

	statuses := make([]domain.TaskStatus, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, *toDomainTaskStatus(row))
	}
	return statuses, nil
}

func (r *TaskStatusRepository) GetByID(ctx context.Context, userID, id string) (*domain.TaskStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	sid, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).GetTaskStatusByID(ctx, sqlc.GetTaskStatusByIDParams{ID: sid, UserID: uid})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTaskStatus(row), nil
}

func (r *TaskStatusRepository) GetBySlug(ctx context.Context, userID, slug string) (*domain.TaskStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).GetTaskStatusBySlug(ctx, sqlc.GetTaskStatusBySlugParams{UserID: uid, Slug: slug})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTaskStatus(row), nil
}

func (r *TaskStatusRepository) GetFirst(ctx context.Context, userID string) (*domain.TaskStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).GetFirstTaskStatus(ctx, uid)
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTaskStatus(row), nil
}

func (r *TaskStatusRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	count, err := r.db.Queries(ctx).CountTaskStatusesByUser(ctx, uid)
	if err != nil {
		return 0, mapError(err)
	}
	return count, nil
}

func (r *TaskStatusRepository) MaxPosition(ctx context.Context, userID string) (int32, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	max, err := r.db.Queries(ctx).MaxTaskStatusPosition(ctx, uid)
	if err != nil {
		return 0, mapError(err)
	}
	return max, nil
}

func (r *TaskStatusRepository) Update(ctx context.Context, s *domain.TaskStatus) (*domain.TaskStatus, error) {
	uid, err := parseUUID(s.UserID)
	if err != nil {
		return nil, err
	}
	sid, err := parseUUID(s.ID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).UpdateTaskStatus(ctx, sqlc.UpdateTaskStatusParams{
		ID:       sid,
		UserID:   uid,
		Name:     s.Name,
		Slug:     s.Slug,
		Position: s.Position,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTaskStatus(row), nil
}

func (r *TaskStatusRepository) SetPosition(ctx context.Context, userID, id string, position int32) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	sid, err := parseUUID(id)
	if err != nil {
		return err
	}

	return mapError(r.db.Queries(ctx).SetTaskStatusPosition(ctx, sqlc.SetTaskStatusPositionParams{
		ID:       sid,
		UserID:   uid,
		Position: position,
	}))
}

func (r *TaskStatusRepository) Delete(ctx context.Context, userID, id string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	sid, err := parseUUID(id)
	if err != nil {
		return err
	}

	return mapError(r.db.Queries(ctx).DeleteTaskStatus(ctx, sqlc.DeleteTaskStatusParams{ID: sid, UserID: uid}))
}

func (r *TaskStatusRepository) IsInUse(ctx context.Context, id string) (bool, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return false, err
	}
	return r.db.Queries(ctx).IsTaskStatusInUse(ctx, sid)
}

func toDomainTaskStatus(s sqlc.TaskStatus) *domain.TaskStatus {
	return &domain.TaskStatus{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		Name:      s.Name,
		Slug:      s.Slug,
		Position:  s.Position,
		IsDefault: s.IsDefault,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

var _ domain.TaskStatusRepository = (*TaskStatusRepository)(nil)
