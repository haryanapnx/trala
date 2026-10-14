package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/repository/postgres/sqlc"
)

type TaskRepository struct {
	db *DB
}

func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	teamID, err := parseUUID(t.TeamID)
	if err != nil {
		return nil, err
	}
	creatorID, err := parseUUID(t.CreatorID)
	if err != nil {
		return nil, err
	}
	statusID, err := parseUUID(t.StatusID)
	if err != nil {
		return nil, err
	}

	assignee := uuid.NullUUID{}
	if t.AssigneeID != nil {
		id, err := parseUUID(*t.AssigneeID)
		if err != nil {
			return nil, err
		}
		assignee = uuid.NullUUID{UUID: id, Valid: true}
	}

	row, err := r.db.Queries(ctx).CreateTask(ctx, sqlc.CreateTaskParams{
		TeamID:      teamID,
		CreatorID:   creatorID,
		AssigneeID:  assignee,
		StatusID:    statusID,
		Title:       t.Title,
		Description: t.Description,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTask(row), nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).GetTaskByID(ctx, tid)
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTask(row), nil
}

func (r *TaskRepository) List(ctx context.Context, f domain.TaskFilter) ([]domain.Task, error) {
	params, err := listParams(f)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Queries(ctx).ListTasks(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, *toDomainTask(row))
	}
	return tasks, nil
}

func (r *TaskRepository) Count(ctx context.Context, f domain.TaskFilter) (int64, error) {
	params, err := listParams(f)
	if err != nil {
		return 0, err
	}

	total, err := r.db.Queries(ctx).CountTasks(ctx, sqlc.CountTasksParams{
		UserID:   params.UserID,
		StatusID: params.StatusID,
		Search:   params.Search,
	})
	if err != nil {
		return 0, mapError(err)
	}
	return total, nil
}

func (r *TaskRepository) Update(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	id, err := parseUUID(t.ID)
	if err != nil {
		return nil, err
	}
	creatorID, err := parseUUID(t.CreatorID)
	if err != nil {
		return nil, err
	}
	statusID, err := parseUUID(t.StatusID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).UpdateTask(ctx, sqlc.UpdateTaskParams{
		ID:          id,
		CreatorID:   creatorID,
		Title:       t.Title,
		Description: t.Description,
		StatusID:    statusID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTask(row), nil
}

func (r *TaskRepository) Assign(ctx context.Context, taskID, creatorID, assigneeID string) (*domain.Task, error) {
	tid, err := parseUUID(taskID)
	if err != nil {
		return nil, err
	}
	cid, err := parseUUID(creatorID)
	if err != nil {
		return nil, err
	}
	aid, err := parseUUID(assigneeID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).AssignTask(ctx, sqlc.AssignTaskParams{
		ID:         tid,
		CreatorID:  cid,
		AssigneeID: uuid.NullUUID{UUID: aid, Valid: true},
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainTask(row), nil
}

func (r *TaskRepository) SoftDelete(ctx context.Context, id, creatorID string) error {
	tid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(creatorID)
	if err != nil {
		return err
	}

	return mapError(r.db.Queries(ctx).SoftDeleteTask(ctx, sqlc.SoftDeleteTaskParams{ID: tid, CreatorID: uid}))
}

// listParams maps a domain filter onto sqlc parameters, turning empty optional
// values into NULL so the query ignores them.
func listParams(f domain.TaskFilter) (sqlc.ListTasksParams, error) {
	userID, err := parseUUID(f.UserID)
	if err != nil {
		return sqlc.ListTasksParams{}, err
	}

	status := uuid.NullUUID{}
	if f.StatusID != "" {
		statusID, err := parseUUID(f.StatusID)
		if err != nil {
			return sqlc.ListTasksParams{}, err
		}
		status = uuid.NullUUID{UUID: statusID, Valid: true}
	}

	var search *string
	if f.Search != "" {
		s := f.Search
		search = &s
	}

	return sqlc.ListTasksParams{
		UserID:     userID,
		StatusID:   status,
		Search:     search,
		PageOffset: f.Offset,
		PageLimit:  f.Limit,
	}, nil
}

func toDomainTask(t sqlc.Task) *domain.Task {
	task := &domain.Task{
		ID:          t.ID.String(),
		TeamID:      t.TeamID.String(),
		CreatorID:   t.CreatorID.String(),
		StatusID:    t.StatusID.String(),
		Title:       t.Title,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		DeletedAt:   t.DeletedAt,
	}
	if t.AssigneeID.Valid {
		assignee := t.AssigneeID.UUID.String()
		task.AssigneeID = &assignee
	}
	return task
}

var _ domain.TaskRepository = (*TaskRepository)(nil)
