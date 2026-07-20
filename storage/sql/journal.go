package sql

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// QueryFromPosition runs a paginated cursor-style SQL query against the given
// query template. The cursor ID appears TWICE in the resulting args slice
// because it fills two distinct roles in the caller's template: it is the
// JOIN key (locating the row whose siblings follow) and the WHERE bound
// (tie-breaking on id when timestamps collide). Pass afterID as the raw
// string form of the cursor.
//
// The kind string ("commands"/"events"/"queries") is interpolated into the
// error message so the diagnostic stays specific without forcing callers to
// duplicate the surrounding boilerplate.
//
// Errors are attributed to the supplied span via cqrsotel.RecordError and
// wrapped with the storage.query_from_position code. The caller owns the
// rows and is responsible for `defer sqlpkg.CloseRows(rows)`.
func QueryFromPosition(
	ctx context.Context,
	db *sql.DB,
	span cqrsotel.Span,
	query string,
	afterID string,
	limit int,
	placeholder string,
	kind string,
) (*sql.Rows, error) {
	args := []any{afterID, afterID}
	query, args = AppendLimit(query, args, limit, placeholder)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query %s from position (limit=%d)", kind, limit))
	}

	return rows, nil
}
