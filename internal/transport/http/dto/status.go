package dto

import (
	"time"

	"github.com/haryanapnx/trala/internal/domain"
)

type CreateStatusRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position *int32 `json:"position"`
}

type UpdateStatusRequest struct {
	Name     *string `json:"name"`
	Slug     *string `json:"slug"`
	Position *int32  `json:"position"`
}

type StatusResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Position  int32     `json:"position"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToStatusResponse(s *domain.TaskStatus) StatusResponse {
	return StatusResponse{
		ID:        s.ID,
		Name:      s.Name,
		Slug:      s.Slug,
		Position:  s.Position,
		IsDefault: s.IsDefault,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func ToStatusResponses(list []domain.TaskStatus) []StatusResponse {
	out := make([]StatusResponse, 0, len(list))
	for i := range list {
		out = append(out, ToStatusResponse(&list[i]))
	}
	return out
}

type StatusListResponse struct {
	Data []StatusResponse `json:"data"`
}
