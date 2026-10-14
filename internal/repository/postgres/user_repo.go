package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/repository/postgres/sqlc"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	row, err := r.db.Queries(ctx).CreateUser(ctx, sqlc.CreateUserParams{
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
	})
	if err != nil {
		mapped := mapError(err)
		if appErr, ok := domain.AsAppError(mapped); ok && appErr.Code == domain.CodeConflict {
			return nil, appErr.WithMessage("email already registered")
		}
		return nil, mapped
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.db.Queries(ctx).GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	row, err := r.db.Queries(ctx).GetUserByID(ctx, uid)
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainUser(row), nil
}

func toDomainUser(u sqlc.User) *domain.User {
	return &domain.User{
		ID:           u.ID.String(),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

// parseUUID converts a string id, mapping a malformed value to not-found so the
// caller cannot distinguish format errors from missing rows.
func parseUUID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, domain.ErrNotFound.WithCause(err)
	}
	return parsed, nil
}

var _ domain.UserRepository = (*UserRepository)(nil)
