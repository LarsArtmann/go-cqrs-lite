package sql

import (
	"context"
	"database/sql"
)

// Executor is the minimal SQL execution interface satisfied by both [*sql.DB]
// and [*sql.Tx]. Stores and helpers that accept an Executor can run their
// operations either directly on the connection pool (auto-commit) or scoped to
// a caller-managed transaction (passed via [*sql.DB.BeginTx]).
//
// Use this to participate in a multi-statement transaction: begin a tx, run
// store operations against it, and commit/rollback as a unit. Stores that hold
// an Executor internally (rather than a bare *sql.DB) gain transactional
// support without changing their method signatures.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
