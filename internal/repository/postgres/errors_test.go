package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryanapnx/trala/internal/domain"
)

func TestMapErrorLockNotAvailableBecomesInProgress(t *testing.T) {
	err := mapError(&pgconn.PgError{Code: "55P03"}) // lock_not_available

	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeInProgress, appErr.Code)
}

func TestMapErrorUniqueViolationBecomesConflict(t *testing.T) {
	err := mapError(&pgconn.PgError{Code: "23505"}) // unique_violation

	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeConflict, appErr.Code)
}

func TestMapErrorForeignKeyViolationBecomesValidation(t *testing.T) {
	err := mapError(&pgconn.PgError{Code: "23503"}) // foreign_key_violation

	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeValidation, appErr.Code)
}
