package adttest_test

import (
	"errors"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestRejectInvalidAggregateFn pins the aggregate-fn whitelist contract that
// the harnesses assert cross-engine: non-constant fn values are rejected with
// the classified metaengine.invalid_aggregate_fn code, constants pass.
func TestRejectInvalidAggregateFn(t *testing.T) {
	err := metaengine.RejectInvalidAggregateFn("DROP TABLE meta_map; --")
	if !errors.Is(err, metaengine.ErrInvalidAggregateFn) {
		t.Fatalf("RejectInvalidAggregateFn hostile value: errors.Is(ErrInvalidAggregateFn) = false, got %v", err)
	}

	if !strings.Contains(err.Error(), "metaengine.invalid_aggregate_fn") {
		t.Fatalf("RejectInvalidAggregateFn hostile value: error does not carry the metaengine.invalid_aggregate_fn code: %v", err)
	}

	for _, fn := range []metaengine.AggregateFn{
		metaengine.AggregateCount,
		metaengine.AggregateSum,
		metaengine.AggregateMin,
		metaengine.AggregateMax,
		metaengine.AggregateAvg,
	} {
		if err := metaengine.RejectInvalidAggregateFn(fn); err != nil {
			t.Errorf("RejectInvalidAggregateFn(%s) = %v, want nil", fn, err)
		}

		if !metaengine.IsValidAggregateFn(fn) {
			t.Errorf("IsValidAggregateFn(%s) = false, want true", fn)
		}
	}

	if metaengine.IsValidAggregateFn("DROP TABLE meta_map; --") {
		t.Error("IsValidAggregateFn(hostile) = true, want false")
	}
}
