package metaengine

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// sharedCountInput mirrors the real-world collision: system.Count (and any
// other counter helper) declares every counter projection on ONE input type.
type sharedCountInput struct{}

type countEventOne struct{ Key string }
type countEventTwo struct{ Key string }

func counterOneQuery() any {
	return Query[sharedCountInput, map[string]int64](
		"counter_one",
		OnRecord(countEventOne{}, func(_ record.Record, _ countEventOne) Delta {
			return Delta{"one": +1}
		}),
	)
}

func counterTwoQuery() any {
	return Query[sharedCountInput, map[string]int64](
		"counter_two",
		OnRecord(countEventTwo{}, func(_ record.Record, _ countEventTwo) Delta {
			return Delta{"two": +1}
		}),
	)
}

// TestExecuteQueryByName_DisambiguatesSharedInputType is the regression test
// for the Count shadowing bug (system/v4 review P1-2): two counter queries on
// one input type. Type dispatch resolves to the most recent registration;
// named dispatch must reach both counters independently.
func TestExecuteQueryByName_DisambiguatesSharedInputType(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		counterOneQuery(),
		counterTwoQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	if err := store.Apply(ctx, "countEventOne", countEventOne{Key: "a"}); err != nil {
		t.Fatalf("apply countEventOne: %v", err)
	}

	if err := store.Apply(ctx, "countEventOne", countEventOne{Key: "a"}); err != nil {
		t.Fatalf("apply countEventOne (2nd): %v", err)
	}

	if err := store.Apply(ctx, "countEventTwo", countEventTwo{Key: "b"}); err != nil {
		t.Fatalf("apply countEventTwo: %v", err)
	}

	one, err := ExecuteTypedByName[sharedCountInput, map[string]int64](
		ctx, store, "counter_one", sharedCountInput{},
	)
	if err != nil {
		t.Fatalf("ExecuteTypedByName counter_one: %v", err)
	}

	if one["one"] != 2 {
		t.Fatalf("counter_one = %v, want one=2", one)
	}

	two, err := ExecuteTypedByName[sharedCountInput, map[string]int64](
		ctx, store, "counter_two", sharedCountInput{},
	)
	if err != nil {
		t.Fatalf("ExecuteTypedByName counter_two: %v", err)
	}

	if two["two"] != 1 {
		t.Fatalf("counter_two = %v, want two=1", two)
	}

	// Documented type-dispatch behavior: the most recently registered query
	// shadows the earlier one for the shared input type.
	typed, err := ExecuteTyped[sharedCountInput, map[string]int64](
		ctx, store, sharedCountInput{},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if len(typed) != 1 || typed["two"] != 1 {
		t.Fatalf("type dispatch = %v, want shadowed {two:1}", typed)
	}
}

func TestExecuteQueryByName_UnknownName(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		counterOneQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	_, err = store.ExecuteQueryByName(context.Background(), "ghost", sharedCountInput{})
	if !errors.Is(err, ErrNoQueryForName) {
		t.Fatalf("ExecuteQueryByName(ghost) error = %v, want ErrNoQueryForName", err)
	}
}

func TestExecuteQueryByName_CancelledContext(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		counterOneQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.ExecuteQueryByName(ctx, "counter_one", sharedCountInput{}); err == nil {
		t.Fatal("ExecuteQueryByName with cancelled context: want error, got nil")
	}
}
