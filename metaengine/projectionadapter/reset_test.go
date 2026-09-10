package projectionadapter_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"log/slog"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Compile-time proof that the Adapter satisfies projectionhost's Resettable
// contract, so a Host.Reset performs a one-call revert of a metaengine-backed
// projection (checkpoint + read-model state cleared together).
var _ projectionhost.Resettable = (*projectionadapter.Adapter)(nil)

func decodeBenchItem(_ string, payload []byte) (any, error) {
	var e benchItem

	err := json.Unmarshal(payload, &e)

	return e, err
}

func itemQuery() metaengine.QueryDecl[findItem, benchItem] {
	return metaengine.Query[findItem, benchItem](
		"find-item",
		metaengine.OnRecord(benchItem{}, func(_ record.Record, e benchItem) (string, benchItem) {
			return e.ID, e
		}),
	)
}

func TestAdapter_Reset_ClearsMemoryBackedStore(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, itemQuery())
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	var logBuf bytes.Buffer
	adapter := projectionadapter.New("items", store, decodeBenchItem,
		projectionadapter.WithLogger(slog.New(slog.NewTextHandler(&logBuf, nil))))

	ctx := context.Background()

	if err := adapter.Handle(
		ctx,
		makeEvent(t, "benchItem", benchItem{ID: "i1", Name: "Widget", Price: 5}),
	); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	reader := metaengine.NewReader[benchItem](store, "find-item")

	before, err := reader.Count(ctx)
	if err != nil {
		t.Fatalf("Count before reset: %v", err)
	}

	if before != 1 {
		t.Fatalf("expected 1 item before reset, got %d", before)
	}

	if err := adapter.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	after, err := reader.Count(ctx)
	if err != nil {
		t.Fatalf("Count after reset: %v", err)
	}

	if after != 0 {
		t.Fatalf("expected 0 items after reset, got %d", after)
	}

	if strings.Contains(logBuf.String(), "could not be bulk-cleared") {
		t.Fatalf(
			"memory engine is clearable; expected no partial-reset warning, got: %s",
			logBuf.String(),
		)
	}
}

// nonResettableEngine is a metaengine.Engine that declares ADTMap support (so
// Plan routes the item query to it) but deliberately does NOT implement
// metaengine.EngineResetter, forcing a partial reset.
type nonResettableEngine struct{}

func (nonResettableEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name: "stub-unclearable",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
}

func (nonResettableEngine) Close() error { return nil }

func TestAdapter_Reset_WarnsAndSucceedsOnUnclearableEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan([]metaengine.Engine{nonResettableEngine{}}, itemQuery())
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	var logBuf bytes.Buffer
	adapter := projectionadapter.New("items", store, decodeBenchItem,
		projectionadapter.WithLogger(slog.New(slog.NewTextHandler(&logBuf, nil))))

	// v4 behavior: a partial reset warns but still returns nil so existing
	// callers that only cleared the checkpoint are not broken. (v5 hardens this
	// to an error.)
	if err := adapter.Reset(context.Background()); err != nil {
		t.Fatalf("Reset on an unclearable engine must return nil in v4, got: %v", err)
	}

	logged := logBuf.String()
	if !strings.Contains(logged, "could not be bulk-cleared") {
		t.Fatalf("expected a partial-reset warning naming the unclearable engine, got: %q", logged)
	}

	if !strings.Contains(logged, "stub-unclearable") {
		t.Fatalf("expected the warning to name the unclearable engine, got: %q", logged)
	}
}
