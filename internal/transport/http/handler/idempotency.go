package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/haryfun/task-management-service/internal/domain"
)

// requireIdempotencyKey validates that the Idempotency-Key header is present and
// a valid UUID. The key's value is consumed by the idempotency replay logic.
func requireIdempotencyKey(r *http.Request) (string, error) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return "", domain.ErrValidation.WithMessage("Idempotency-Key header is required")
	}
	if _, err := uuid.Parse(key); err != nil {
		return "", domain.ErrValidation.WithMessage("Idempotency-Key must be a valid UUID")
	}
	return key, nil
}
