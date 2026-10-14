package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/transport/http/response"
)

// Recover turns a panic into a 500 response in the standard error envelope. The
// stack trace is logged server-side and never sent to the client.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.ErrorContext(r.Context(), "panic recovered",
						"request_id", RequestIDFrom(r.Context()),
						"panic", fmt.Sprint(rec),
						"stack", string(debug.Stack()),
					)
					response.Write(w, http.StatusInternalServerError,
						domain.CodeInternal, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
