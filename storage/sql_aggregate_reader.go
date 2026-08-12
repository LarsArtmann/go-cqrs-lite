package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

const (
	defaultListingPageSize = 20
	maxListingPageSize     = 100
)

// SQLStreamReader queries a projection table maintained by StreamProjection.
type SQLStreamReader struct {
	db      *sql.DB
	dialect sqlpkg.Dialect
	table   listingTable
}

var _ listing.StreamReader = (*SQLStreamReader)(nil)

// NewSQLStreamReader creates a reader that queries the streams projection table.
func NewSQLStreamReader(
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
) (*SQLStreamReader, error) {
	tbl, err := newListingTable(tablePrefix)
	if err != nil {
		return nil, errorfamily.NewRejection("listing.invalid_prefix",
			"invalid table prefix")
	}

	return &SQLStreamReader{
		db:      db,
		dialect: dialect,
		table:   tbl,
	}, nil
}

func (r *SQLStreamReader) List(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	return listing.ListRefsFromStatus(r, ctx, opts)
}

func (r *SQLStreamReader) ListWithStatus(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.StreamStatus], error) {
	if opts.Type == "" {
		return nil, errorfamily.NewRejection(
			"listing.type_required",
			"ListOptions.Type is required",
		)
	}

	query, args, limit := r.buildListQuery(opts)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "listing.sql_list", "listing sql list")
	}
	defer sqlpkg.CloseRows(rows)

	items, err := scanStreamStatuses(rows)
	if err != nil {
		return nil, err
	}

	return paginateStatuses(items, limit), nil
}

func (r *SQLStreamReader) buildListQuery(opts listing.ListOptions) (string, []any, uint) {
	var (
		conditions []string
		args       []any
	)

	pi := 1

	conditions = append(conditions, "aggregate_type = "+r.dialect.Placeholder(pi))
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
		conditions = append(conditions, "aggregate_id > "+r.dialect.Placeholder(pi))
		args = append(args, opts.After.String())
		pi++
	}

	limit := opts.Limit
	if limit == 0 {
		limit = defaultListingPageSize
	}

	query := fmt.Sprintf(
		"SELECT aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status FROM %s WHERE %s ORDER BY aggregate_id LIMIT %s",
		r.table.name,
		strings.Join(conditions, " AND "),
		r.dialect.Placeholder(pi),
	)

	args = append(args, limit+1)

	return query, args, limit
}

func scanStreamStatuses(rows *sql.Rows) ([]listing.StreamStatus, error) {
	var items []listing.StreamStatus

	for rows.Next() {
		var (
			streamType string
			streamID   string
			version    int
			count      uint
			lastAt     string
			statusInt  int
		)

		err := rows.Scan(&streamType, &streamID, &version, &count, &lastAt, &statusInt)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err, "listing.sql_scan", "listing sql scan")
		}

		parsedID, err := id.ParseStreamID(streamID)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"listing.sql_parse_id",
				"listing sql parse id",
			)
		}

		items = append(items, listing.StreamStatus{
			Ref: listing.StreamListing{
				ID:         parsedID,
				Type:       id.StreamType(streamType),
				Version:    event.Version(version),
				EventCount: count,
			},
			Status: event.TombstoneStatus(statusInt),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "listing.sql_rows", "listing sql rows")
	}

	return items, nil
}

func paginateStatuses(
	items []listing.StreamStatus,
	limit uint,
) *listing.Page[listing.StreamStatus] {
	hasMore := uint(len(items)) > limit
	if hasMore {
		items = items[:limit]
	}

	return &listing.Page[listing.StreamStatus]{Items: items, HasMore: hasMore}
}

// Deprecated: use SQLStreamReader.
type SQLAggregateReader = SQLStreamReader

// Deprecated: use NewSQLStreamReader.
func NewSQLAggregateReader(
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
) (*SQLStreamReader, error) {
	return NewSQLStreamReader(db, tablePrefix, dialect)
}
