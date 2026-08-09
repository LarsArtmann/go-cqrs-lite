package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── Get: point lookup ───

// ErrNotFound is returned by Get when the key does not exist.
var ErrNotFound = errors.New("system: not found")

// Get performs a point lookup on a Lookup projection by key.
// Returns [ErrNotFound] when the key does not exist.
//
//	task, err := system.Get[TaskView](ctx, sys, "get-task", taskID)
func Get[R any](ctx context.Context, sys *System, name, key string) (R, error) {
	store := sys.MetaEngine()
	if store == nil {
		var zero R

		return zero, ErrNoProjections
	}

	reader := metaengine.NewReader[R](store, name)
	v, found, err := reader.Get(ctx, key)
	if err != nil {
		var zero R

		return zero, fmt.Errorf("system: get %q: %w", name, err)
	}

	if !found {
		var zero R

		return zero, ErrNotFound
	}

	return v, nil
}

// ─── Find: filtered scan ───

// SortOrder controls sort direction for [OrderBy].
type SortOrder bool

const (
	Asc  SortOrder = false // ascending
	Desc SortOrder = true  // descending
)

// FindOption configures a [Find] query.
type FindOption func(*findConfig)

type findConfig struct {
	filters []filterCond
	sortBy  string
	desc    bool
	limit   int
	cursor  any
}

type filterCond struct {
	field string
	value any
}

// Where filters results by field=value (equality).
// The field must be declared in [QuerySet.Filterable] at build time
// for SQL index pushdown; undeclared fields fall back to in-Go filtering.
func Where(field string, value any) FindOption {
	return func(c *findConfig) {
		c.filters = append(c.filters, filterCond{field: field, value: value})
	}
}

// OrderBy sorts results by field in the given direction.
func OrderBy(field string, order SortOrder) FindOption {
	return func(c *findConfig) {
		c.sortBy = field
		c.desc = bool(order)
	}
}

// Limit caps the number of results returned.
func Limit(n int) FindOption {
	return func(c *findConfig) { c.limit = n }
}

// After sets a pagination cursor from a previous page's [PaginatedResult.Cursor].
// Use with [Limit] to paginate through large result sets.
func After(cursor string) FindOption {
	return func(c *findConfig) { c.cursor = cursor }
}

// Find queries a QuerySet projection with optional filters, sort, and limit.
// Returns a slice of results; empty slice if no matches.
//
//	// All tasks
//	all, err := system.Find[TaskView](ctx, sys, "tasks")
//
//	// Filtered + sorted + limited
//	hot, err := system.Find[TaskView](ctx, sys, "tasks",
//	    system.Where("status", "active"),
//	    system.OrderBy("priority", system.Desc),
//	    system.Limit(10))
func Find[R any](ctx context.Context, sys *System, name string, opts ...FindOption) ([]R, error) {
	store := sys.MetaEngine()
	if store == nil {
		return nil, ErrNoProjections
	}

	cfg := findConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	var scanOpts []metaengine.ScanOption

	for _, f := range cfg.filters {
		scanOpts = append(scanOpts, metaengine.WithFilter(f.field, metaengine.FilterEq, f.value))
	}

	if cfg.sortBy != "" {
		scanOpts = append(scanOpts, metaengine.WithSort(cfg.sortBy, cfg.desc))
	}

	if cfg.limit > 0 {
		scanOpts = append(scanOpts, metaengine.WithLimit(cfg.limit))
	}

	if cfg.cursor != nil {
		if s, ok := cfg.cursor.(string); ok && s != "" {
			scanOpts = append(scanOpts, metaengine.WithCursorString(s))
		}
	}

	reader := metaengine.NewReader[R](store, name)

	result, err := reader.Scan(ctx, scanOpts...)
	if err != nil {
		return nil, fmt.Errorf("system: find %q: %w", name, err)
	}

	return result, nil
}

// ─── GetCount: counter read ───

// GetCount reads all counter keys and values from a Count projection.
//
//	counts, err := system.GetCount(ctx, sys, "task-counts")
//	fmt.Println(counts["active"]) // int64
func GetCount(ctx context.Context, sys *System, name string) (map[string]int64, error) {
	store := sys.MetaEngine()
	if store == nil {
		return nil, ErrNoProjections
	}

	result, err := metaengine.ExecuteTyped[CountInput, map[string]int64](
		ctx, store, CountInput{},
	)
	if err != nil {
		return nil, fmt.Errorf("system: count %q: %w", name, err)
	}

	return result, nil
}
