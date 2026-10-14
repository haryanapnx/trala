package domain

import (
	"context"
	"time"
)

// User is an account. PasswordHash never leaves the persistence layer's control
// of the use cases; it is still carried here for verification.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRepository is the persistence port for users.
type UserRepository interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}
