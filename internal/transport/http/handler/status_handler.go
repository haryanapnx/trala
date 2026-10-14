package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/model"
	"github.com/haryfun/task-management-service/internal/transport/http/dto"
	"github.com/haryfun/task-management-service/internal/transport/http/middleware"
	"github.com/haryfun/task-management-service/internal/transport/http/response"
)

// StatusUsecase is the slice of the status use case the handler needs.
type StatusUsecase interface {
	List(ctx context.Context, userID string) ([]domain.TaskStatus, error)
	Create(ctx context.Context, in model.CreateStatusInput) (*domain.TaskStatus, error)
	Update(ctx context.Context, in model.UpdateStatusInput) (*domain.TaskStatus, error)
	Delete(ctx context.Context, userID, statusID string) error
}

type StatusHandler struct {
	statuses StatusUsecase
}

func NewStatusHandler(statuses StatusUsecase) *StatusHandler {
	return &StatusHandler{statuses: statuses}
}

// @Summary      List statuses
// @Tags         statuses
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.StatusListResponse
// @Router       /statuses [get]
func (h *StatusHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	list, err := h.statuses.List(r.Context(), userID)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.StatusListResponse{Data: dto.ToStatusResponses(list)})
}

// @Summary      Create a status step
// @Tags         statuses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateStatusRequest true "Status payload"
// @Success      201 {object} dto.StatusResponse
// @Failure      400 {object} response.ErrorBody
// @Failure      409 {object} response.ErrorBody
// @Router       /statuses [post]
func (h *StatusHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.CreateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	status, err := h.statuses.Create(r.Context(), model.CreateStatusInput{
		UserID:   userID,
		Name:     req.Name,
		Slug:     req.Slug,
		Position: req.Position,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.ToStatusResponse(status))
}

// @Summary      Update or reorder a status step
// @Tags         statuses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Status ID"
// @Param        request body dto.UpdateStatusRequest true "Fields to update"
// @Success      200 {object} dto.StatusResponse
// @Failure      400 {object} response.ErrorBody
// @Failure      404 {object} response.ErrorBody
// @Router       /statuses/{id} [put]
func (h *StatusHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.UpdateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	status, err := h.statuses.Update(r.Context(), model.UpdateStatusInput{
		UserID:   userID,
		StatusID: chi.URLParam(r, "id"),
		Name:     req.Name,
		Slug:     req.Slug,
		Position: req.Position,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ToStatusResponse(status))
}

// @Summary      Delete a status step
// @Tags         statuses
// @Security     BearerAuth
// @Param        id path string true "Status ID"
// @Success      204 "No Content"
// @Failure      409 {object} response.ErrorBody
// @Router       /statuses/{id} [delete]
func (h *StatusHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		response.Error(w, r, domain.ErrUnauthorized)
		return
	}

	if err := h.statuses.Delete(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		response.Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
