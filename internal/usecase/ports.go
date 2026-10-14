package usecase

import (
	"context"

	"github.com/haryfun/task-management-service/internal/domain"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenIssuer issues access tokens for authenticated users.
type TokenIssuer interface {
	Issue(userID string) (token string, expiresIn int64, err error)
}

// TaskResponseEncoder serializes a task into the exact bytes returned to the
// client (status + body). The use case needs this to persist a replayable
// response; the wire shape itself stays in the transport layer.
type TaskResponseEncoder func(task *domain.Task) (status int, body []byte, err error)

// AssignmentNotification describes a task assignment to be delivered.
type AssignmentNotification struct {
	TaskID     string
	TaskTitle  string
	FromUserID string
	ToUserID   string
}

// Notifier delivers notifications about task changes. A failing Notify must
// abort the surrounding operation, so implementations should be cheap and
// reliable (the mock just logs).
type Notifier interface {
	NotifyAssigned(ctx context.Context, notification AssignmentNotification) error
}
