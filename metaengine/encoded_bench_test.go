package metaengine

import (
	"context"
	"encoding/json/jsontext"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type benchFoldEvent struct {
	TaskID string
	Name   string
}

// benchFoldStore plans a one-query store and returns it with the query meta
// and its insert fold, so the benchmarks exercise the real applyFold funnel.
func benchFoldStore(b *testing.B) (*Store, queryMeta, Fold) {
	b.Helper()

	store, err := Plan([]Engine{NewMemoryEngine()}, Query[benchFoldEvent, map[string]string](
		"bench_fold_tasks",
		On(benchFoldEvent{}, func(e benchFoldEvent) (string, string) {
			return e.TaskID, e.Name
		}),
	))
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = store.Close() })

	q, ok := store.queries["bench_fold_tasks"]
	if !ok {
		b.Fatal("bench query not registered")
	}

	folds := q.QueryFolds()
	if len(folds) == 0 {
		b.Fatal("no folds registered")
	}

	return store, q, folds[0]
}

// The number that defends the encoded-apply fix: the struct hot path pays
// only the jsontext.Value type assertion in decodeRawFoldPayload, while the
// encoded path additionally pays the per-fold JSON decode. Run both and read
// the delta — the struct number must stay in the low nanoseconds with zero
// allocations.
func BenchmarkApplyFoldStructPayload(b *testing.B) {
	store, q, fold := benchFoldStore(b)

	ctx := context.Background()
	rec := record.Record{Type: fold.EventType()}
	payload := benchFoldEvent{TaskID: "t1", Name: "bench"}

	b.ReportAllocs()

	for b.Loop() {
		if err := store.applyFold(ctx, q, fold, rec, payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplyFoldEncodedPayload(b *testing.B) {
	store, q, fold := benchFoldStore(b)

	ctx := context.Background()
	rec := record.Record{Type: fold.EventType()}
	payload := jsontext.Value(`{"TaskID":"t1","Name":"bench"}`)

	b.ReportAllocs()

	for b.Loop() {
		if err := store.applyFold(ctx, q, fold, rec, payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeRawFoldPayloadStruct isolates the funnel's struct-path cost:
// one failed interface assertion, no decode, no allocation.
func BenchmarkDecodeRawFoldPayloadStruct(b *testing.B) {
	_, _, fold := benchFoldStore(b)

	payload := benchFoldEvent{TaskID: "t1", Name: "bench"}

	b.ReportAllocs()

	for b.Loop() {
		decoded, err := decodeRawFoldPayload(fold, payload)
		if err != nil {
			b.Fatal(err)
		}

		if _, ok := decoded.(benchFoldEvent); !ok {
			b.Fatalf("struct payload must pass through unchanged, got %T", decoded)
		}
	}
}
