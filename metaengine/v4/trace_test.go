package metaengine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func newTraceStore(t *testing.T) (*Store, Engine) {
	t.Helper()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store, eng
}

func TestTrace_RecordAndRead(t *testing.T) {
	t.Parallel()

	store, _ := newTraceStore(t)

	var buf bytes.Buffer
	rec := RecordTrace(store, &buf)

	for i := range 5 {
		item := roleItemCreated{ID: fmt.Sprintf("t%d", i), Name: fmt.Sprintf("name%d", i)}
		if err := store.Apply(context.Background(), "roleItemCreated", item); err != nil {
			t.Fatal(err)
		}
	}

	for i := range 2 {
		if _, err := ExecuteTyped[roleFindItem, roleItem](
			context.Background(), store, roleFindItem{ID: fmt.Sprintf("t%d", i)},
		); err != nil {
			t.Fatal(err)
		}
	}

	rec.Close()

	ops, err := ReadTrace(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	if len(ops) != 7 {
		t.Fatalf("want 7 ops, got %d", len(ops))
	}

	for _, op := range ops {
		if op.V != traceVersion {
			t.Fatalf("op %q missing version stamp: %+v", op.Op, op)
		}
	}

	summary := TraceStats(ops)
	if summary.Applies != 5 || summary.Queries != 2 {
		t.Fatalf("want 5 applies + 2 queries, got %+v", summary)
	}

	if summary.ByName["roleItemCreated"] != 5 || summary.ByName["role_items"] != 2 {
		t.Fatalf("unexpected name counts: %+v", summary.ByName)
	}
}

func TestTrace_ReplayIntoFreshStore(t *testing.T) {
	t.Parallel()

	source, _ := newTraceStore(t)

	var buf bytes.Buffer
	rec := RecordTrace(source, &buf)

	for i := range 4 {
		item := roleItemCreated{ID: fmt.Sprintf("r%d", i), Name: fmt.Sprintf("n%d", i)}
		if err := source.Apply(context.Background(), "roleItemCreated", item); err != nil {
			t.Fatal(err)
		}
	}

	rec.Close()

	ops, err := ReadTrace(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	target, targetEng := newTraceStore(t)

	sink := StoreTraceSink(
		target,
		func(eventType string, seq int) any {
			return roleItemCreated{ID: fmt.Sprintf("r%d", seq), Name: fmt.Sprintf("n%d", seq)}
		},
		func(name string, seq int) any {
			return roleFindItem{ID: fmt.Sprintf("r%d", seq)}
		},
	)

	if err := ReplayTrace(context.Background(), ops, sink); err != nil {
		t.Fatal(err)
	}

	if got := mirrorRows(t, targetEng, "role_items"); got != 4 {
		t.Fatalf("replay should rebuild 4 rows, got %d", got)
	}
}

func TestTrace_ChainsExistingHooks(t *testing.T) {
	t.Parallel()

	store, _ := newTraceStore(t)

	applied := 0
	folded := 0

	WithHooks(store, Hooks{
		OnApply: func(string, time.Duration, error) { applied++ },
		OnFold:  func(string, string, FoldKind, time.Duration, error) { folded++ },
	})

	var buf bytes.Buffer
	rec := RecordTrace(store, &buf)

	if err := store.Apply(
		context.Background(),
		"roleItemCreated",
		roleItemCreated{ID: "c", Name: "c"},
	); err != nil {
		t.Fatal(err)
	}

	rec.Close()

	if applied != 1 {
		t.Fatalf("previous OnApply hook should still fire, fired %d times", applied)
	}

	if folded != 1 {
		t.Fatalf("previous OnFold hook should still fire, fired %d times", folded)
	}

	if !strings.Contains(buf.String(), "roleItemCreated") {
		t.Fatalf("trace should contain the apply: %q", buf.String())
	}
}

func TestTrace_CloseStopsRecording(t *testing.T) {
	t.Parallel()

	store, _ := newTraceStore(t)

	var buf bytes.Buffer
	rec := RecordTrace(store, &buf)

	_ = store.Apply(context.Background(), "roleItemCreated", roleItemCreated{ID: "a", Name: "a"})

	rec.Close()

	_ = store.Apply(context.Background(), "roleItemCreated", roleItemCreated{ID: "b", Name: "b"})

	content := buf.String()
	if strings.Count(content, "\n") != 1 {
		t.Fatalf(
			"expected exactly 1 recorded line after Close, got %d",
			strings.Count(content, "\n"),
		)
	}
}

// TestTrace_SurfacesEncodeError proves a failing writer is reported via
// TraceRecorder.Err instead of silently dropping trace lines.
func TestTrace_SurfacesEncodeError(t *testing.T) {
	t.Parallel()

	store, _ := newTraceStore(t)

	rec := RecordTrace(store, errWriter{})

	if err := store.Apply(
		context.Background(),
		"roleItemCreated",
		roleItemCreated{ID: "e", Name: "e"},
	); err != nil {
		t.Fatal(err)
	}

	if rec.Err() == nil {
		t.Fatal("expected Err() to surface the writer failure")
	}

	rec.Close()
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestTrace_ReplaySkipsUnknownOps(t *testing.T) {
	t.Parallel()

	ops := []TraceOp{
		{Op: "future-op", Name: "whatever"},
		{Op: TraceOpApply, Name: "roleItemCreated"},
	}

	target, targetEng := newTraceStore(t)

	sink := StoreTraceSink(
		target,
		func(_ string, seq int) any {
			return roleItemCreated{ID: fmt.Sprintf("u%d", seq), Name: "u"}
		},
		func(string, int) any { return roleFindItem{ID: "u0"} },
	)

	if err := ReplayTrace(context.Background(), ops, sink); err != nil {
		t.Fatal(err)
	}

	if got := mirrorRows(t, targetEng, "role_items"); got != 1 {
		t.Fatalf("want 1 row, got %d", got)
	}
}
