package metaengine_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// fakeAggregateEngine embeds fakeEngine and adds stub implementations of all
// five aggregate pushdown interfaces. Doctor's aggregateCapabilities checks
// type assertions — the methods need to exist but never execute in these tests.
type fakeAggregateEngine struct {
	*fakeEngine
}

func (e *fakeAggregateEngine) Aggregate(
	_ context.Context, _ string, _ metaengine.AggregateFn, _ string, _ []metaengine.FilterSpec,
) (float64, error) {
	return 0, nil
}

func (e *fakeAggregateEngine) GroupedAggregate(
	_ context.Context, _ string, _ metaengine.AggregateFn, _ string, _ string, _ []metaengine.FilterSpec,
) (map[string]float64, error) {
	return nil, nil
}

func (e *fakeAggregateEngine) MultiAggregate(
	_ context.Context, _ string, _ []metaengine.AggregateSpec, _ []metaengine.FilterSpec,
) (map[string]float64, error) {
	return nil, nil
}

func (e *fakeAggregateEngine) MultiGroupedAggregate(
	_ context.Context, _ string, _ []metaengine.AggregateSpec, _ string, _ []metaengine.FilterSpec,
) ([]metaengine.GroupedAggregateRow, error) {
	return nil, nil
}

func (e *fakeAggregateEngine) DistinctValues(
	_ context.Context, _ string, _ string, _ []metaengine.FilterSpec,
) ([]any, error) {
	return nil, nil
}

func TestDoctor_AggregatePushdownSection_WithCapabilities(t *testing.T) {
	t.Parallel()

	engine := &fakeAggregateEngine{
		fakeEngine: &fakeEngine{profile: metaengine.EngineProfile{
			Name:    "agg-engine",
			NsPerOp: 1000,
			Supports: map[metaengine.ADT]metaengine.Complexity{
				metaengine.ADTMap: metaengine.ComplexityO1,
			},
		}},
	}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.Doctor(t.Context())

	for _, want := range []string{
		"--- Aggregate Pushdown ---",
		"find_task",
		"pushdown: scalar, grouped, multi, multi-grouped, distinct",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Doctor: expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestDoctor_AggregatePushdownSection_NoneForMemoryEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.Doctor(t.Context())

	if !strings.Contains(output, "--- Aggregate Pushdown ---") {
		t.Errorf("Doctor: expected aggregate pushdown header, got:\n%s", output)
	}

	// Memory engine does NOT implement any aggregate interfaces.
	if !strings.Contains(output, "--- Aggregate Pushdown ---\n  none") {
		t.Errorf("Doctor: expected 'none' for non-aggregate engine, got:\n%s", output)
	}
}
