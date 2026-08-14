package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Inserter encapsulates the entity-specific bits needed to append rows to a
// SQL journal (Save / AppendBatch) for one CQRS message type (commands,
// queries). It is the write-side counterpart of JournalReader: construct one
// per call site via a small helper on the store, then call Insert or
// InsertAll inside the caller-owned transaction (RunInTx).
//
// The receiver types still own their method-level wrappers so they satisfy
// command.Store, query.Store, etc.
type Inserter[T any] struct {
	Dialect Dialect
	Table   string
	// Columns are the INSERT column names, in RowArgs order.
	Columns []string

	// RowArgs converts one item into the ordered INSERT argument slice,
	// including metadata marshalling. The returned error is wrapped as
	// Corruption with MarshalErrCode by Insert/InsertAll.
	RowArgs func(T) ([]any, error)

	// Describe renders one item for error messages, e.g.
	// "command CreateTask for Task/1". Used in every wrap this type emits.
	Describe func(T) string

	// EntityNoun is the singular noun used in error messages ("command", "query").
	EntityNoun string

	// MarshalErrCode is the errorfamily code for RowArgs failures.
	MarshalErrCode string
	// InsertErrCode is the errorfamily code for INSERT statement failures.
	InsertErrCode string

	// Duplicate converts a duplicate-key failure into the entity-specific
	// conflict error (e.g. command.ErrDuplicateCommand). It receives the
	// offending item so the message can name its ID.
	Duplicate func(err error, item T) error
}

// Insert appends a single row. Duplicate-key violations are routed to the
// Duplicate hook; every other failure is wrapped as Infrastructure with
// InsertErrCode. The transaction is owned by the caller.
func (in *Inserter[T]) Insert(ctx context.Context, tx *sql.Tx, item T) error {
	args, err := in.RowArgs(item)
	if err != nil {
		return errorfamily.WrapCorruption(err, in.MarshalErrCode,
			fmt.Sprintf("marshal row for %s %s", in.EntityNoun, in.Describe(item)))
	}

	query := in.insertSQL()
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if IsDuplicateKeyError(err) {
			return in.Duplicate(err, item)
		}

		return errorfamily.WrapInfrastructure(err, in.InsertErrCode,
			fmt.Sprintf("insert %s %s", in.EntityNoun, in.Describe(item)))
	}

	return nil
}

// InsertAll appends every row, stopping at the first failure so the whole
// batch rolls back with the caller's transaction. It deliberately inserts
// row-by-row instead of one multi-VALUES statement: command and query batches
// are small, and per-row inserts keep duplicate errors naming the offending
// ID. High-volume event batches keep their dedicated chunked multi-VALUES
// path in SharedBatchInsertEvents.
func (in *Inserter[T]) InsertAll(ctx context.Context, tx *sql.Tx, items []T) error {
	for _, item := range items {
		if err := in.Insert(ctx, tx, item); err != nil {
			return err
		}
	}

	return nil
}

func (in *Inserter[T]) insertSQL() string {
	placeholders := make([]string, len(in.Columns))
	for i := range in.Columns {
		placeholders[i] = in.Dialect.Placeholder(i + 1)
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		in.Table,
		strings.Join(in.Columns, ", "),
		strings.Join(placeholders, ", "),
	)
}
