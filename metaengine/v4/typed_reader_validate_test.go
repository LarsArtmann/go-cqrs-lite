package metaengine_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// recordingAggregateEngine counts how often an aggregate entry point reaches
// the engine. The methods are never meant to run: every test asserts calls==0.
type recordingAggregateEngine struct {
	*fakeEngine

	calls atomic.Int32
}

func (e *recordingAggregateEngine) hit() { e.calls.Add(1) }

func (e *recordingAggregateEngine) Aggregate(
	_ context.Context, _ string, _ metaengine.AggregateFn, _ string, _ []metaengine.FilterSpec,
) (float64, error) {
	e.hit()

	return 0, nil
}

func (e *recordingAggregateEngine) GroupedAggregate(
	_ context.Context, _ string, _ metaengine.AggregateFn, _, _ string, _ []metaengine.FilterSpec,
) (map[string]float64, error) {
	e.hit()

	return nil, nil
}

func (e *recordingAggregateEngine) MultiAggregate(
	_ context.Context, _ string, _ []metaengine.AggregateSpec, _ []metaengine.FilterSpec,
) (map[string]float64, error) {
	e.hit()

	return nil, nil
}

func (e *recordingAggregateEngine) MultiGroupedAggregate(
	_ context.Context, _ string, _ []metaengine.AggregateSpec, _ string, _ []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	e.hit()

	return nil, nil
}

func (e *recordingAggregateEngine) DistinctValues(
	_ context.Context, _ string, _ string, _ []metaengine.FilterSpec,
) ([]any, error) {
	e.hit()

	return nil, nil
}

// aggregateCalls lists every TypedReader entry point that pushes filters to
// an engine without going through Scan.
func aggregateCalls(
	r *metaengine.TypedReader[FindTaskResult],
) map[string]func(context.Context, ...metaengine.ScanOption) error {
	specs := []metaengine.AggregateSpec{{Fn: metaengine.AggregateSum, Column: "Priority"}}

	return map[string]func(context.Context, ...metaengine.ScanOption) error{
		"Count": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.Count(c, opts...)

			return err
		},
		"Sum": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.Sum(c, "Priority", opts...)

			return err
		},
		"Min": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.Min(c, "Priority", opts...)

			return err
		},
		"Max": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.Max(c, "Priority", opts...)

			return err
		},
		"Avg": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.Avg(c, "Priority", opts...)

			return err
		},
		"Distinct": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.Distinct(c, "Title", opts...)

			return err
		},
		"GroupedCount": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.GroupedCount(c, "Status", opts...)

			return err
		},
		"GroupedSum": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.GroupedSum(c, "Priority", "Status", opts...)

			return err
		},
		"MultiAggregate": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.MultiAggregate(c, specs, opts...)

			return err
		},
		"MultiGroupedAggregate": func(c context.Context, opts ...metaengine.ScanOption) error {
			_, err := r.MultiGroupedAggregate(c, specs, "Status", opts...)

			return err
		},
	}
}

func newAggregatePushdownStore(t *testing.T) (
	*recordingAggregateEngine,
	*metaengine.TypedReader[FindTaskResult],
) {
	t.Helper()

	eng := &recordingAggregateEngine{fakeEngine: &fakeEngine{profile: metaengine.EngineProfile{
		Name:    "recording-agg",
		NsPerOp: 1000,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return eng, metaengine.NewReader[FindTaskResult](store, "find_task")
}

// TestAggregatePushdownValidatesFilters pins: an operator outside the defined
// FilterOp constants, or a column that is not an identifier, is rejected
// before ANY aggregate entry point calls the engine. Operators and columns
// are spliced into SQL text, so an unvalidated one is an injection surface.
func TestAggregatePushdownValidatesFilters(t *testing.T) {
	t.Parallel()

	eng, reader := newAggregatePushdownStore(t)
	ctx := context.Background()

	type case_ struct {
		opts  []metaengine.ScanOption
		where string
	}

	for name, call := range aggregateCalls(reader) {
		for _, bad := range []case_{
			{
				opts:  []metaengine.ScanOption{metaengine.WithFilter("Status", "1=1; DROP TABLE meta_map; --", "x")},
				where: "operator",
			},
			{
				opts:  []metaengine.ScanOption{metaengine.WithFilter("Status') OR 1=1 --", metaengine.FilterEq, "x")},
				where: "filter column",
			},
			{
				opts:  []metaengine.ScanOption{metaengine.WithRange("Status') OR 1=1 --", "a", "b")},
				where: "range column",
			},
		} {
			err := call(ctx, bad.opts...)
			if err == nil {
				t.Errorf("%s with a bad %s must fail", name, bad.where)

				continue
			}

			if got := eng.calls.Load(); got != 0 {
				t.Fatalf("%s: engine was called %d times before validation failed", name, got)
			}
		}
	}
}

// TestAggregatePushdownValidatesColumns pins the value and group-by column
// arguments of the aggregate entry points: they are interpolated into SQL.
func TestAggregatePushdownValidatesColumns(t *testing.T) {
	t.Parallel()

	eng, reader := newAggregatePushdownStore(t)
	ctx := context.Background()

	calls := map[string]func(context.Context) error{
		"Sum": func(c context.Context) error {
			_, err := reader.Sum(c, "Priority') OR 1=1 --")

			return err
		},
		"Distinct": func(c context.Context) error {
			_, err := reader.Distinct(c, "Title') OR 1=1 --")

			return err
		},
		"GroupedCount": func(c context.Context) error {
			_, err := reader.GroupedCount(c, "Status') OR 1=1 --")

			return err
		},
		"MultiAggregate": func(c context.Context) error {
			_, err := reader.MultiAggregate(c, []metaengine.AggregateSpec{{
				Fn: metaengine.AggregateSum, Column: "Priority') OR 1=1 --",
			}})

			return err
		},
		"MultiGroupedAggregate": func(c context.Context) error {
			_, err := reader.MultiGroupedAggregate(c, []metaengine.AggregateSpec{{
				Fn: metaengine.AggregateCount,
			}}, "Status') OR 1=1 --")

			return err
		},
	}

	for name, call := range calls {
		if err := call(ctx); err == nil {
			t.Errorf("%s with an injected column must fail", name)
		}
	}

	if got := eng.calls.Load(); got != 0 {
		t.Fatalf("engine was called %d times", got)
	}
}

// TestScanFilterInTypedSlice pins WithFilter(FilterIn, []T): a typed slice is
// coerced to the list every path evaluates. Before, only []any worked: the
// closure path matched nothing and SQL engines dropped the clause.
func TestScanFilterInTypedSlice(t *testing.T) {
	t.Parallel()

	_, reader := seededTaskReader(t)
	ctx := context.Background()

	rows, err := reader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterIn, []string{"open"}),
		metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("FilterIn over []string = %d rows, want 3", len(rows))
	}

	n, err := reader.Count(ctx,
		metaengine.WithFilter("Status", metaengine.FilterIn, []string{"closed"}))
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	if n != 2 {
		t.Fatalf("Count with FilterIn over []string = %d, want 2", n)
	}

	_, err = reader.Scan(ctx, metaengine.WithFilter("Status", metaengine.FilterIn, "open"))
	if err == nil || !strings.Contains(err.Error(), "FilterIn") {
		t.Fatalf("FilterIn with a non-slice value must be rejected, got: %v", err)
	}
}
