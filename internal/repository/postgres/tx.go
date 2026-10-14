package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/repository/postgres/sqlc"
)

type txContextKey struct{}

// DB owns the connection pool and exposes sqlc queries that transparently
// participate in an active transaction when one is present in the context.
type DB struct {
	pool        *pgxpool.Pool
	lockTimeout time.Duration
}

func NewDB(pool *pgxpool.Pool, lockTimeout time.Duration) *DB {
	return &DB{pool: pool, lockTimeout: lockTimeout}
}

// Queries returns a query set bound to the context's transaction, or to the
// pool when no transaction is active.
func (db *DB) Queries(ctx context.Context) *sqlc.Queries {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return sqlc.New(tx)
	}
	return sqlc.New(db.pool)
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// WithinTransaction runs fn in a transaction, committing on nil error and
// rolling back otherwise.
func (db *DB) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// Safe no-op after a successful commit.
	defer func() { _ = tx.Rollback(ctx) }()

	// Bound lock waits: if this transaction has to wait on a lock held by
	// another request (e.g. the same idempotency key in flight) longer than the
	// timeout, Postgres aborts it with lock_not_available, which the repository
	// maps to a 409 "in progress" instead of hanging.
	if db.lockTimeout > 0 {
		if _, err := tx.Exec(ctx, "SELECT set_config('lock_timeout', $1, true)", db.lockTimeout.String()); err != nil {
			return fmt.Errorf("set lock timeout: %w", err)
		}
	}

	if err := fn(context.WithValue(ctx, txContextKey{}, tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

var _ domain.TxManager = (*DB)(nil)
