package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/listing"
)

const (
	defaultListingPageSize = 20
	maxListingPageSize     = 100
)

// SQLAggregateReader queries a projection table maintained by AggregateProjection.
type SQLAggregateReader struct {
	db        *sql.DB
	tableName string
}

var _ listing.AggregateReader = (*SQLAggregateReader)(nil)

// NewSQLAggregateReader creates a reader that queries the aggregates projection table.
func NewSQLAggregateReader(db *sql.DB, tablePrefix string) (*SQLAggregateReader, error) {
	err := validateListingTablePrefix(tablePrefix)
	if err != nil {
		return nil, fmt.Errorf("invalid table prefix %q: %w", tablePrefix, err)
	}

	return &SQLAggregateReader{
		db:        db,
		tableName: tablePrefix + "listing_aggregates",
	}, nil
}

func (r *SQLAggregateReader) List(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.AggregateRef], error) {
	return listRefsFromStatus(r, ctx, opts)
}

func (r *SQLAggregateReader) ListWithStatus(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.AggregateStatus], error) {
	if opts.Type == "" {
		return nil, event.NewRejection(
			"listing.type_required",
			"ListOptions.Type is required",
		)
	}

	var (
		conditions []string
		args       []any
	)

	conditions = append(conditions, "aggregate_type = ?")
	args = append(args, string(opts.Type))

	switch opts.Tombstone {
	case listing.TombstoneExclude:
		conditions = append(conditions, "tombstone_status = 0")
	case listing.TombstoneOnly:
		conditions = append(conditions, "tombstone_status = 1")
	}

	if !opts.After.IsZero() {
		conditions = append(conditions, "aggregate_id > ?")
		args = append(args, opts.After.String())
	}

	limit := opts.Limit
	if limit == 0 {
		limit = defaultListingPageSize
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
			"listing.sql_list",
			"listing sql list",
		)
	}
	defer func() { _ = rows.Close() }()

	var items []listing.AggregateStatus

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
				"listing.sql_scan",
				"listing sql scan",
			)
		}

		parsedID, err := id.ParseAggregateID(aggID)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"listing.sql_parse_id",
				"listing sql parse id",
			)
		}

		items = append(items, listing.AggregateStatus{
			Ref: listing.AggregateRef{
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
			"listing.sql_rows",
			"listing sql rows",
		)
	}

	hasMore := uint(len(items)) > limit
	if hasMore {
		items = items[:limit]
	}

	return &listing.Page[listing.AggregateStatus]{Items: items, HasMore: hasMore}, nil
}

func listRefsFromStatus(
	r listing.AggregateReader,
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.AggregateRef], error) {
	statusPage, err := r.ListWithStatus(ctx, opts)
	if err != nil {
		return nil, err
	}

	refs := make([]listing.AggregateRef, len(statusPage.Items))
	for i, s := range statusPage.Items {
		refs[i] = s.Ref
	}

	return &listing.Page[listing.AggregateRef]{Items: refs, HasMore: statusPage.HasMore}, nil
}
