package notifier

import (
	"context"
	"log/slog"

	"github.com/haryanapnx/trala/internal/usecase"
)

// Mock is a stand-in notifier that emits a structured log line instead of
// contacting an external service.
type Mock struct {
	log *slog.Logger
}

func NewMock(log *slog.Logger) *Mock {
	return &Mock{log: log}
}

func (m *Mock) NotifyAssigned(ctx context.Context, n usecase.AssignmentNotification) error {
	m.log.InfoContext(ctx, "notification: task assigned",
		"task_id", n.TaskID,
		"title", n.TaskTitle,
		"from_user", n.FromUserID,
		"to_user", n.ToUserID,
	)
	return nil
}

var _ usecase.Notifier = (*Mock)(nil)
