package response

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryfun/task-management-service/internal/domain"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func TestErrorMapsDomainCodes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"validation", domain.ErrValidation.WithMessage("title is required"), http.StatusBadRequest, "VALIDATION_ERROR"},
		{"not found", domain.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"conflict", domain.ErrConflict, http.StatusConflict, "CONFLICT"},
		{"idempotency reused", domain.ErrIdempotencyReused, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED"},
		{"internal", domain.ErrInternal, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			Error(rr, httptest.NewRequest(http.MethodGet, "/", nil), tc.err)

			assert.Equal(t, tc.status, rr.Code)
			assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

			var body ErrorBody
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.Equal(t, tc.status, body.Status)
			assert.NotEmpty(t, body.Message)
			assert.NotEmpty(t, body.Timestamp)
		})
	}
}

func TestErrorTreatsUnknownAsInternalAndHidesDetails(t *testing.T) {
	rr := httptest.NewRecorder()
	Error(rr, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("db connection refused"))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db connection refused")

	var body ErrorBody
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body.Code)
	assert.Equal(t, "internal server error", body.Message)
}
