package dto

import (
	"time"

	"github.com/haryanapnx/trala/internal/domain"
)

type CreateTaskRequest struct {
	TeamID      string  `json:"team_id"`
	StatusID    string  `json:"status_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	StatusID    *string `json:"status_id"`
}

type AssignTaskRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type TaskResponse struct {
	ID          string    `json:"id"`
	TeamID      string    `json:"team_id"`
	CreatorID   string    `json:"creator_id"`
	AssigneeID  *string   `json:"assignee_id"`
	StatusID    string    `json:"status_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToTaskResponse(t *domain.Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		TeamID:      t.TeamID,
		CreatorID:   t.CreatorID,
		AssigneeID:  t.AssigneeID,
		StatusID:    t.StatusID,
		Title:       t.Title,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func ToTaskResponses(list []domain.Task) []TaskResponse {
	out := make([]TaskResponse, 0, len(list))
	for i := range list {
		out = append(out, ToTaskResponse(&list[i]))
	}
	return out
}

type PageMeta struct {
	Page       int32 `json:"page"`
	Limit      int32 `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int32 `json:"total_pages"`
}

type TaskListResponse struct {
	Data []TaskResponse `json:"data"`
	Meta PageMeta       `json:"meta"`
}
