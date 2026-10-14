package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/transport/http/response"
)

// TokenVerifier validates an access token and returns its subject (user id).
type TokenVerifier interface {
	Parse(token string) (string, error)
}

type userIDKey struct{}

// Auth rejects requests without a valid bearer token and stores the
// authenticated user id in the request context.
func Auth(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				response.Error(w, r, domain.ErrUnauthorized.WithMessage("missing or malformed authorization header"))
				return
			}

			userID, err := verifier.Parse(token)
			if err != nil {
				response.Error(w, r, domain.ErrUnauthorized.WithMessage("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFrom returns the authenticated user id stored in ctx.
func UserIDFrom(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey{}).(string)
	return id, ok
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
