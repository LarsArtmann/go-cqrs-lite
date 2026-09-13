package metaengine

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"sync/atomic"
	"testing"
)

// streamingScanTestEngine embeds the concrete memory engine (promoting all
// of its backend interfaces) and adds the StreamingScan capability. When
// cannedRows is non-nil it yields exactly those rows (proving the streaming
// path was taken); otherwise it delegates to MapScan so output stays
// faithful.
type streamingScanTestEngine struct {
	*memoryEngine
	calls      atomic.Int32
	cannedRows []any
}

func newStreamingScanTestEngine() *streamingScanTestEngine {
	return &streamingScanTestEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine),
	} //nolint:forcetypeassert // concrete memory engine, internal test
}

func (e *streamingScanTestEngine) StreamScan(
	ctx context.Context,
	collection string,
	_ []FilterSpec,
	_ *SortSpec,
) iter.Seq2[any, error] {
	e.calls.Add(1)

	return func(yield func(any, error) bool) {
		if e.cannedRows != nil {
			for _, row := range e.cannedRows {
				if !yield(row, nil) {
					return
				}
			}

			return
		}

		result, err := e.memoryEngine.MapScan(ctx, collection, nil, nil, nil, 0)
		if err != nil {
			yield(nil, err)

			return
		}

		for _, row := range result.Items {
			if !yield(row, nil) {
				return
			}
		}
	}
}

func TestStreamCollection_FallbackScansAll(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	ctx := context.Background()

	if err := store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "A", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "B", Status: "open"}},
	}); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	var rows []any

	if err := store.StreamCollection(ctx, "tasks", func(row any) error {
		rows = append(rows, row)

		return nil
	}); err != nil {
		t.Fatalf("StreamCollection: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestStreamCollection_PrefersStreamingScan(t *testing.T) {
	t.Parallel()

	eng := &streamingScanTestEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine), //nolint:forcetypeassert // concrete memory engine, internal test
		cannedRows:   []any{"CANARY"},
	}

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	var rows []any

	if err := store.StreamCollection(context.Background(), "tasks", func(row any) error {
		rows = append(rows, row)

		return nil
	}); err != nil {
		t.Fatalf("StreamCollection: %v", err)
	}

	if got := eng.calls.Load(); got != 1 {
		t.Fatalf("expected 1 StreamScan call, got %d", got)
	}

	if len(rows) != 1 || rows[0] != "CANARY" {
		t.Fatalf("expected canned streaming row, got %#v", rows)
	}
}

func TestStreamCollection_UnknownCollection(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	err := store.StreamCollection(context.Background(), "nope", func(any) error { return nil })
	if !errors.Is(err, errCollectionNotFound) {
		t.Fatalf("expected errCollectionNotFound, got %v", err)
	}
}

func TestStreamCollection_PropagatesFnError(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	ctx := context.Background()

	if err := store.ApplyBatch(ctx, []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "A", Status: "open"}},
	}); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	sentinel := errors.New("stop streaming")

	err := store.StreamCollection(ctx, "tasks", func(any) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error unchanged, got %v", err)
	}
}

func TestExport_UsesStreamingScanAndMatchesFallbackOutput(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plainStore := newMemoryTestStore(t)
	eng := newStreamingScanTestEngine()

	streamStore, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	events := []EventInput{
		{Type: "task_created", Payload: testTask{ID: "t1", Title: "A", Status: "open"}},
		{Type: "task_created", Payload: testTask{ID: "t2", Title: "B", Status: "done"}},
	}

	for _, store := range []*Store{plainStore, streamStore} {
		if err := store.ApplyBatch(ctx, events); err != nil {
			t.Fatalf("ApplyBatch: %v", err)
		}
	}

	var plainBuf, streamBuf bytes.Buffer

	if err := plainStore.Export(ctx, &plainBuf); err != nil {
		t.Fatalf("plain Export: %v", err)
	}

	if err := streamStore.Export(ctx, &streamBuf); err != nil {
		t.Fatalf("stream Export: %v", err)
	}

	if eng.calls.Load() == 0 {
		t.Fatal("expected Export to use StreamingScan")
	}

	if plainBuf.String() != streamBuf.String() {
		t.Fatalf(
			"export output diverged:\nplain:  %s\nstream: %s",
			plainBuf.String(),
			streamBuf.String(),
		)
	}
}
