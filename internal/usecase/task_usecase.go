package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/model"
)

// idempotencyWindow is how long a stored response is replayed for the same key.
const idempotencyWindow = 24 * time.Hour

// errIdempotencyConflict means a concurrent request claimed the key first. It
// forces a rollback so this request's task is discarded, after which the
// winner's stored response is replayed.
var errIdempotencyConflict = errors.New("idempotency key claimed concurrently")

type TaskUsecase struct {
	tasks       domain.TaskRepository
	statuses    domain.TaskStatusRepository
	teams       domain.TeamRepository
	taskLogs    domain.TaskLogRepository
	idempotency domain.IdempotencyRepository
	notifier    Notifier
	tx          domain.TxManager
	encode      TaskResponseEncoder
}

func NewTaskUsecase(
	tasks domain.TaskRepository,
	statuses domain.TaskStatusRepository,
	teams domain.TeamRepository,
	idempotency domain.IdempotencyRepository,
	taskLogs domain.TaskLogRepository,
	notifier Notifier,
	tx domain.TxManager,
	encode TaskResponseEncoder,
) *TaskUsecase {
	return &TaskUsecase{
		tasks:       tasks,
		statuses:    statuses,
		teams:       teams,
		taskLogs:    taskLogs,
		idempotency: idempotency,
		notifier:    notifier,
		tx:          tx,
		encode:      encode,
	}
}

// Create creates a task. With an idempotency key it also stores the response in
// the same transaction and replays it for retries within the window.
func (u *TaskUsecase) Create(ctx context.Context, in model.CreateTaskInput) (*model.CreateTaskResult, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, domain.ErrValidation.WithMessage("title is required")
	}
	if strings.TrimSpace(in.TeamID) == "" {
		return nil, domain.ErrValidation.WithMessage("team_id is required")
	}

	key := strings.TrimSpace(in.IdempotencyKey)
	requestHash := hashCreateRequest(in)

	var out *model.CreateTaskResult
	err := u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if key != "" {
			replayed, err := u.resolveIdempotency(txCtx, key, requestHash)
			if err != nil {
				return err
			}
			if replayed != nil {
				out = replayed
				return nil
			}
		}

		task, err := u.createTask(txCtx, in)
		if err != nil {
			return err
		}

		status, body, err := u.encode(task)
		if err != nil {
			return domain.ErrInternal.WithCause(err)
		}

		if key != "" {
			_, err := u.idempotency.Create(txCtx, &domain.IdempotencyRecord{
				Key:            key,
				UserID:         in.UserID,
				RequestHash:    requestHash,
				ResponseStatus: int16(status),
				ResponseBody:   body,
			})
			if err != nil {
				if appErr, ok := domain.AsAppError(err); ok && appErr.Code == domain.CodeConflict {
					return errIdempotencyConflict
				}
				return err
			}
		}

		out = &model.CreateTaskResult{Task: task, StatusCode: status, Body: body}
		return nil
	})

	if errors.Is(err, errIdempotencyConflict) {
		// Our task was rolled back; replay the response stored by the winner.
		record, getErr := u.idempotency.Get(ctx, key)
		if getErr != nil {
			return nil, getErr
		}
		if record.RequestHash != requestHash {
			return nil, domain.ErrIdempotencyReused
		}
		return &model.CreateTaskResult{StatusCode: int(record.ResponseStatus), Body: record.ResponseBody, Replayed: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// resolveIdempotency returns a stored response to replay, or nil to proceed with
// creating a new task.
func (u *TaskUsecase) resolveIdempotency(ctx context.Context, key, requestHash string) (*model.CreateTaskResult, error) {
	record, err := u.idempotency.Get(ctx, key)
	if err != nil {
		if appErr, ok := domain.AsAppError(err); ok && appErr.Code == domain.CodeNotFound {
			return nil, nil
		}
		return nil, err
	}

	if time.Since(record.CreatedAt) < idempotencyWindow {
		if record.RequestHash != requestHash {
			return nil, domain.ErrIdempotencyReused
		}
		return &model.CreateTaskResult{StatusCode: int(record.ResponseStatus), Body: record.ResponseBody, Replayed: true}, nil
	}

	// Expired: the key may be used as a new request.
	if err := u.idempotency.Delete(ctx, key); err != nil {
		return nil, err
	}
	return nil, nil
}

func (u *TaskUsecase) createTask(ctx context.Context, in model.CreateTaskInput) (*domain.Task, error) {
	member, err := u.teams.IsMember(ctx, in.TeamID, in.UserID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, domain.ErrForbidden.WithMessage("you are not a member of this team")
	}

	statusID, err := u.resolveStatus(ctx, in.UserID, in.StatusID)
	if err != nil {
		return nil, err
	}

	return u.tasks.Create(ctx, &domain.Task{
		TeamID:      in.TeamID,
		CreatorID:   in.UserID,
		StatusID:    statusID,
		Title:       in.Title,
		Description: in.Description,
	})
}

// hashCreateRequest fingerprints the meaningful request fields so a reused key
// with a different payload can be detected.
func hashCreateRequest(in model.CreateTaskInput) string {
	payload := struct {
		UserID      string  `json:"user_id"`
		TeamID      string  `json:"team_id"`
		StatusID    string  `json:"status_id"`
		Title       string  `json:"title"`
		Description *string `json:"description"`
	}{
		UserID:      in.UserID,
		TeamID:      in.TeamID,
		StatusID:    in.StatusID,
		Title:       in.Title,
		Description: in.Description,
	}

	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (u *TaskUsecase) Get(ctx context.Context, userID, taskID string) (*domain.Task, error) {
	task, err := u.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if !canAccess(task, userID) {
		// Not-found rather than forbidden, so task existence is not leaked.
		return nil, domain.ErrNotFound
	}
	return task, nil
}

// canAccess reports whether the user may read the task: they are its creator or
// its assignee.
func canAccess(task *domain.Task, userID string) bool {
	if task.CreatorID == userID {
		return true
	}
	return task.AssigneeID != nil && *task.AssigneeID == userID
}

// resolveStatus returns the requested status (validated as belonging to the
// user) or, when none is given, the user's first status by position.
func (u *TaskUsecase) resolveStatus(ctx context.Context, userID, statusID string) (string, error) {
	if strings.TrimSpace(statusID) != "" {
		status, err := u.statuses.GetByID(ctx, userID, statusID)
		if err != nil {
			if appErr, ok := domain.AsAppError(err); ok && appErr.Code == domain.CodeNotFound {
				return "", domain.ErrValidation.WithMessage("status_id is invalid")
			}
			return "", err
		}
		return status.ID, nil
	}

	status, err := u.statuses.GetFirst(ctx, userID)
	if err != nil {
		return "", err
	}
	return status.ID, nil
}

const (
	defaultPageSize int32 = 10
	maxPageSize     int32 = 100
)

// List returns the user's tasks (as creator or assignee) with optional status
// and title filters, plus pagination metadata.
func (u *TaskUsecase) List(ctx context.Context, in model.ListTasksInput) (*model.TaskListResult, error) {
	page, limit := normalizePagination(in.Page, in.Limit)

	filter := domain.TaskFilter{
		UserID: in.UserID,
		Search: strings.TrimSpace(in.Search),
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	if slug := strings.TrimSpace(in.Status); slug != "" {
		status, err := u.statuses.GetBySlug(ctx, in.UserID, slug)
		if err != nil {
			if appErr, ok := domain.AsAppError(err); ok && appErr.Code == domain.CodeNotFound {
				// Unknown status slug yields an empty page rather than an error.
				return emptyPage(page, limit), nil
			}
			return nil, err
		}
		filter.StatusID = status.ID
	}

	tasks, err := u.tasks.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	total, err := u.tasks.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &model.TaskListResult{
		Tasks:      tasks,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
	}, nil
}

func emptyPage(page, limit int32) *model.TaskListResult {
	return &model.TaskListResult{Tasks: []domain.Task{}, Page: page, Limit: limit}
}

func normalizePagination(page, limit int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	return page, limit
}

func totalPages(total int64, limit int32) int32 {
	if limit <= 0 {
		return 0
	}
	pages := total / int64(limit)
	if total%int64(limit) != 0 {
		pages++
	}
	return int32(pages)
}

// Update applies a partial update to a task the user created.
func (u *TaskUsecase) Update(ctx context.Context, in model.UpdateTaskInput) (*domain.Task, error) {
	existing, err := u.tasks.GetByID(ctx, in.TaskID)
	if err != nil {
		return nil, err
	}
	if existing.CreatorID != in.UserID {
		// Not-found rather than forbidden, so task existence is not leaked.
		return nil, domain.ErrNotFound
	}

	updated := *existing

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, domain.ErrValidation.WithMessage("title cannot be empty")
		}
		updated.Title = title
	}
	if in.Description != nil {
		updated.Description = in.Description
	}
	if in.StatusID != nil {
		if strings.TrimSpace(*in.StatusID) == "" {
			return nil, domain.ErrValidation.WithMessage("status_id cannot be empty")
		}
		statusID, err := u.resolveStatus(ctx, in.UserID, *in.StatusID)
		if err != nil {
			return nil, err
		}
		updated.StatusID = statusID
	}

	return u.tasks.Update(ctx, &updated)
}

// Delete soft-deletes a task the user created.
func (u *TaskUsecase) Delete(ctx context.Context, userID, taskID string) error {
	existing, err := u.tasks.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if existing.CreatorID != userID {
		return domain.ErrNotFound
	}
	return u.tasks.SoftDelete(ctx, taskID, userID)
}

// Assign reassigns a task to a team member (or the creator themself). Updating
// the assignee, writing the audit log and notifying all happen in one
// transaction: a failure in any step rolls the whole operation back.
func (u *TaskUsecase) Assign(ctx context.Context, in model.AssignTaskInput) (*domain.Task, error) {
	if strings.TrimSpace(in.AssigneeID) == "" {
		return nil, domain.ErrValidation.WithMessage("assignee_id is required")
	}

	var assigned *domain.Task
	err := u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		task, err := u.tasks.GetByID(txCtx, in.TaskID)
		if err != nil {
			return err
		}
		if task.CreatorID != in.UserID {
			return domain.ErrNotFound
		}

		member, err := u.teams.IsMember(txCtx, task.TeamID, in.AssigneeID)
		if err != nil {
			return err
		}
		if !member {
			return domain.ErrForbidden.WithMessage("assignee is not a member of this team")
		}

		updated, err := u.tasks.Assign(txCtx, task.ID, in.UserID, in.AssigneeID)
		if err != nil {
			return err
		}

		field := "assignee_id"
		newValue := in.AssigneeID
		var oldValue string
		if task.AssigneeID != nil {
			oldValue = *task.AssigneeID
		}

		if _, err := u.taskLogs.Create(txCtx, &domain.TaskLog{
			TaskID:   task.ID,
			ActorID:  in.UserID,
			Action:   "assigned",
			Field:    &field,
			OldValue: &oldValue,
			NewValue: &newValue,
		}); err != nil {
			return err
		}

		if err := u.notifier.NotifyAssigned(txCtx, AssignmentNotification{
			TaskID:     updated.ID,
			TaskTitle:  updated.Title,
			FromUserID: in.UserID,
			ToUserID:   in.AssigneeID,
		}); err != nil {
			return domain.ErrInternal.WithCause(err)
		}

		assigned = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return assigned, nil
}
