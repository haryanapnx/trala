package domain

import "context"

// TxManager runs fn inside a single database transaction: it commits when fn
// returns nil and rolls back otherwise. The transaction is carried in the
// context so repositories automatically participate in it.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
