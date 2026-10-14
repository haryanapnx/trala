package domain

import (
	"context"
	"time"
)

// IdempotencyRecord stores the response produced for a request key so that
// retries within the replay window can be answered identically.
type IdempotencyRecord struct {
	Key            string
	UserID         string
	RequestHash    string
	ResponseStatus int16
	ResponseBody   []byte
	CreatedAt      time.Time
}

// IdempotencyRepository is the persistence port for idempotency records. Create
// returns ErrConflict when the key already exists (a concurrent duplicate).
type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Create(ctx context.Context, record *IdempotencyRecord) (*IdempotencyRecord, error)
	Delete(ctx context.Context, key string) error
}
