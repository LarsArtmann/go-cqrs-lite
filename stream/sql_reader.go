package stream

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLAggregateReader queries a projection table maintained by AggregateProjection.
// The table must be created and kept in sync separately.
type SQLAggregateReader struct {
	db        *sql.DB
	tableName string
}

var _ AggregateReader = (*SQLAggregateReader)(nil)

// NewSQLAggregateReader creates a reader that queries the aggregates projection table.
func NewSQLAggregateReader(db *sql.DB, tablePrefix string) (*SQLAggregateReader, error) {
	if err := validateTablePrefix(tablePrefix); err != nil {
		return nil, err
	}

	return &SQLAggregateReader{
		db:        db,
		tableName: tablePrefix + "stream_aggregates",
	}, nil
}

func (r *SQLAggregateReader) List(
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateRef], error) {
	statusPage, err := r.ListWithStatus(ctx, opts)
	if err != nil {
		return nil, err
	}

	refs := make([]AggregateRef, len(statusPage.Items))
	for i, s := range statusPage.Items {
		refs[i] = s.Ref
	}

	return &Page[AggregateRef]{Items: refs, HasMore: statusPage.HasMore}, nil
}

func (r *SQLAggregateReader) ListWithStatus(
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateStatus], error) {
	if opts.Type == "" {
		return nil, event.NewRejection(
			"stream.type_required",
			"ListOptions.Type is required",
		)
	}

	var conditions []string
	var args []any

	conditions = append(conditions, "aggregate_type = ?")
	args = append(args, string(opts.Type))

	switch opts.Tombstone {
	case TombstoneExclude:
		conditions = append(conditions, "tombstone_status = 0")
	case TombstoneOnly:
		conditions = append(conditions, "tombstone_status = 1")
	}

	if !opts.After.IsZero() {
		conditions = append(conditions, "aggregate_id > ?")
		args = append(args, opts.After.String())
	}

	limit := opts.Limit
	if limit == 0 {
		limit = defaultPageSize
	}

	query := fmt.Sprintf(
		"SELECT aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status FROM %s WHERE %s ORDER BY aggregate_id LIMIT ?",
		r.tableName,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"stream.sql_list",
			"stream sql list",
		)
	}
	defer func() { _ = rows.Close() }()

	var items []AggregateStatus

	for rows.Next() {
		var (
			aggType   string
			aggID     string
			version   int
			count     uint
			lastAt    string
			statusInt int
		)

		err = rows.Scan(&aggType, &aggID, &version, &count, &lastAt, &statusInt)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"stream.sql_scan",
				"stream sql scan",
			)
		}

		parsedID, err := id.ParseAggregateID(aggID)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"stream.sql_parse_id",
				"stream sql parse id",
			)
		}

		items = append(items, AggregateStatus{
			Ref: AggregateRef{
				ID:         parsedID,
				Type:       event.AggregateType(aggType),
				Version:    event.Version(version),
				EventCount: count,
			},
			Status: event.TombstoneStatus(statusInt),
		})
	}

	if err = rows.Err(); err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"stream.sql_rows",
			"stream sql rows",
		)
	}

	hasMore := uint(len(items)) > limit
	if hasMore {
		items = items[:limit]
	}

	return &Page[AggregateStatus]{Items: items, HasMore: hasMore}, nil
}
