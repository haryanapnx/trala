package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/haryanapnx/trala/internal/domain"
)

// mapError converts driver-level errors into domain errors. Unknown errors are
// returned as-is and surface as 500s at the transport layer.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return domain.ErrConflict.WithCause(err)
		case "23503": // foreign_key_violation
			return domain.ErrValidation.WithMessage("referenced resource does not exist").WithCause(err)
		case "23514": // check_violation
			return domain.ErrValidation.WithCause(err)
		case "55P03": // lock_not_available (lock_timeout exceeded)
			return domain.ErrInProgress.WithCause(err)
		}
	}

	return err
}
