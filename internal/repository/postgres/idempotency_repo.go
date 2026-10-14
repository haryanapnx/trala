package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/repository/postgres/sqlc"
)

type IdempotencyRepository struct {
	db *DB
}

func NewIdempotencyRepository(db *DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (r *IdempotencyRepository) Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	row, err := r.db.Queries(ctx).GetIdempotencyKey(ctx, key)
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainIdempotency(row), nil
}

// Create inserts a record. A key collision (concurrent duplicate) is reported as
// ErrConflict so the caller can replay the winner's response.
func (r *IdempotencyRepository) Create(ctx context.Context, record *domain.IdempotencyRecord) (*domain.IdempotencyRecord, error) {
	userID, err := parseUUID(record.UserID)
	if err != nil {
		return nil, err
	}

	row, err := r.db.Queries(ctx).CreateIdempotencyKey(ctx, sqlc.CreateIdempotencyKeyParams{
		Key:            record.Key,
		UserID:         userID,
		RequestHash:    record.RequestHash,
		ResponseStatus: record.ResponseStatus,
		ResponseBody:   string(record.ResponseBody),
	})
	if err != nil {
		// ON CONFLICT DO NOTHING yields no row when the key already exists.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConflict
		}
		return nil, mapError(err)
	}
	return toDomainIdempotency(row), nil
}

func (r *IdempotencyRepository) Delete(ctx context.Context, key string) error {
	return mapError(r.db.Queries(ctx).DeleteIdempotencyKey(ctx, key))
}

func toDomainIdempotency(k sqlc.IdempotencyKey) *domain.IdempotencyRecord {
	return &domain.IdempotencyRecord{
		Key:            k.Key,
		UserID:         k.UserID.String(),
		RequestHash:    k.RequestHash,
		ResponseStatus: k.ResponseStatus,
		ResponseBody:   []byte(k.ResponseBody),
		CreatedAt:      k.CreatedAt,
	}
}

var _ domain.IdempotencyRepository = (*IdempotencyRepository)(nil)
