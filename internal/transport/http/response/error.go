package response

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/haryfun/task-management-service/internal/domain"
)

// ErrorBody is the single, consistent error envelope for every 4xx/5xx response.
type ErrorBody struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// statusForCode maps a domain error code to its HTTP status.
func statusForCode(code domain.ErrorCode) int {
	switch code {
	case domain.CodeValidation:
		return http.StatusBadRequest
	case domain.CodeNotFound:
		return http.StatusNotFound
	case domain.CodeUnauthorized:
		return http.StatusUnauthorized
	case domain.CodeForbidden:
		return http.StatusForbidden
	case domain.CodeConflict, domain.CodeIdempotencyReused, domain.CodeInProgress:
		return http.StatusConflict
	case domain.CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	default:
		return http.StatusInternalServerError
	}
}

// Error renders err as the standard error envelope. Errors that are not domain
// errors are treated as internal: their details are logged but never exposed.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := domain.AsAppError(err)
	if !ok {
		appErr = domain.ErrInternal.WithCause(err)
	}

	status := statusForCode(appErr.Code)
	if appErr.Code == domain.CodeInProgress {
		// Tell the client the same key is being processed and to retry shortly.
		w.Header().Set("Retry-After", "1")
	}
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed",
			"code", appErr.Code,
			"error", appErr.Err,
		)
	}

	Write(w, status, appErr.Code, appErr.Message)
}

// Write emits the error envelope without logging (used by panic recovery, which
// logs the stack itself).
func Write(w http.ResponseWriter, status int, code domain.ErrorCode, message string) {
	JSON(w, status, ErrorBody{
		Status:    status,
		Code:      string(code),
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
