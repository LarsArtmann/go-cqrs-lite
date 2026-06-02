package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

const (
	defaultListingPageSize = 20
	maxListingPageSize     = 100
)

// SQLAggregateReader queries a projection table maintained by AggregateProjection.
type SQLAggregateReader struct {
	db        *sql.DB
	dialect   sqlpkg.Dialect
	tableName string
}

var _ listing.AggregateReader = (*SQLAggregateReader)(nil)

// NewSQLAggregateReader creates a reader that queries the aggregates projection table.
func NewSQLAggregateReader(
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
) (*SQLAggregateReader, error) {
	err := validateListingTablePrefix(tablePrefix)
	if err != nil {
		return nil, event.NewRejection("listing.invalid_prefix",
			"invalid table prefix")
	}

	return &SQLAggregateReader{
		db:        db,
		dialect:   dialect,
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

	pi := 1

	conditions = append(conditions, fmt.Sprintf("aggregate_type = %s", r.dialect.Placeholder(pi)))
	args = append(args, string(opts.Type))
	pi++

	switch opts.Tombstone {
	case listing.TombstoneExclude:
		conditions = append(conditions, "tombstone_status = 0")
	case listing.TombstoneOnly:
		conditions = append(conditions, "tombstone_status = 1")
	case listing.TombstoneInclude:
	}

	if !opts.After.IsZero() {
		conditions = append(conditions, fmt.Sprintf("aggregate_id > %s", r.dialect.Placeholder(pi)))
		args = append(args, opts.After.String())
		pi++
	}

	limit := opts.Limit
	if limit == 0 {
		limit = defaultListingPageSize
	}

	query := fmt.Sprintf(
		"SELECT aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status FROM %s WHERE %s ORDER BY aggregate_id LIMIT %s",
		r.tableName,
		strings.Join(conditions, " AND "),
		r.dialect.Placeholder(pi),
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
