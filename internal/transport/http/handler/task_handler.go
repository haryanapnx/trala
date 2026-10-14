package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/model"
	"github.com/haryanapnx/trala/internal/transport/http/dto"
	"github.com/haryanapnx/trala/internal/transport/http/middleware"
	"github.com/haryanapnx/trala/internal/transport/http/response"
)

// TaskUsecase is the slice of the task use case the handler needs.
type TaskUsecase interface {
	Create(ctx context.Context, in model.CreateTaskInput) (*model.CreateTaskResult, error)
	Get(ctx context.Context, userID, taskID string) (*domain.Task, error)
	List(ctx context.Context, in model.ListTasksInput) (*model.TaskListResult, error)
	Update(ctx context.Context, in model.UpdateTaskInput) (*domain.Task, error)
	Delete(ctx context.Context, userID, taskID string) error
	Assign(ctx context.Context, in model.AssignTaskInput) (*domain.Task, error)
}

type TaskHandler struct {
	tasks TaskUsecase
}

func NewTaskHandler(tasks TaskUsecase) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

// Create handles POST /v1/tasks. The Idempotency-Key header is required; retries
// with the same key replay the first response.
//
// @Summary      Create a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key header string true "UUID idempotency key"
// @Param        request body dto.CreateTaskRequest true "Task payload"
// @Success      201 {object} dto.TaskResponse
// @Failure      400 {object} response.ErrorBody
// @Failure      403 {object} response.ErrorBody
// @Failure      409 {object} response.ErrorBody
// @Router       /tasks [post]
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	key, err := requireIdempotencyKey(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.CreateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	result, err := h.tasks.Create(r.Context(), model.CreateTaskInput{
		UserID:         userID,
		TeamID:         req.TeamID,
		StatusID:       req.StatusID,
		Title:          req.Title,
		Description:    req.Description,
		IdempotencyKey: key,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	if result.Replayed {
		w.Header().Set("X-Idempotent-Replayed", "true")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(result.StatusCode)
	_, _ = w.Write(result.Body)
}

// Get handles GET /v1/tasks/{id}.
//
// @Summary      Get a task
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task ID"
// @Success      200 {object} dto.TaskResponse
// @Failure      404 {object} response.ErrorBody
// @Router       /tasks/{id} [get]
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	task, err := h.tasks.Get(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ToTaskResponse(task))
}

// List handles GET /v1/tasks with optional status/search filters and pagination.
//
// @Summary      List tasks
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Status slug"
// @Param        search query string false "Case-insensitive title search"
// @Param        page   query int    false "Page number" default(1)
// @Param        limit  query int    false "Page size (max 100)" default(10)
// @Success      200 {object} dto.TaskListResponse
// @Router       /tasks [get]
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	query := r.URL.Query()
	result, err := h.tasks.List(r.Context(), model.ListTasksInput{
		UserID: userID,
		Status: query.Get("status"),
		Search: query.Get("search"),
		Page:   parseQueryInt(query.Get("page"), 0),
		Limit:  parseQueryInt(query.Get("limit"), 0),
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.TaskListResponse{
		Data: dto.ToTaskResponses(result.Tasks),
		Meta: dto.PageMeta{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// Update handles PUT /v1/tasks/{id}.
//
// @Summary      Update a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task ID"
// @Param        request body dto.UpdateTaskRequest true "Fields to update"
// @Success      200 {object} dto.TaskResponse
// @Failure      400 {object} response.ErrorBody
// @Failure      404 {object} response.ErrorBody
// @Router       /tasks/{id} [put]
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.UpdateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	task, err := h.tasks.Update(r.Context(), model.UpdateTaskInput{
		UserID:      userID,
		TaskID:      chi.URLParam(r, "id"),
		Title:       req.Title,
		Description: req.Description,
		StatusID:    req.StatusID,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ToTaskResponse(task))
}

// Delete handles DELETE /v1/tasks/{id} (soft delete).
//
// @Summary      Delete a task
// @Tags         tasks
// @Security     BearerAuth
// @Param        id path string true "Task ID"
// @Success      204 "No Content"
// @Failure      404 {object} response.ErrorBody
// @Router       /tasks/{id} [delete]
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	if err := h.tasks.Delete(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		response.Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Assign handles POST /v1/tasks/{id}/assign.
//
// @Summary      Assign a task to a team member
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task ID"
// @Param        request body dto.AssignTaskRequest true "Assignee"
// @Success      200 {object} dto.TaskResponse
// @Failure      403 {object} response.ErrorBody
// @Failure      404 {object} response.ErrorBody
// @Router       /tasks/{id}/assign [post]
func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.AssignTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	task, err := h.tasks.Assign(r.Context(), model.AssignTaskInput{
		UserID:     userID,
		TaskID:     chi.URLParam(r, "id"),
		AssigneeID: req.AssigneeID,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ToTaskResponse(task))
}
