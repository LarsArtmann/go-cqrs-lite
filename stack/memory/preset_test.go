package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	stackmemory "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// todoKey is a branded string key for read-model tests.
type todoKey string

func (k todoKey) String() string { return string(k) }

type todoView struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func TestNew_ProducesFullyWiredBundle(t *testing.T) {
	t.Parallel()

	b, err := stackmemory.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	checks := []struct {
		name string
		ok   bool
	}{
		{"EventSink", b.EventSink != nil},
		{"EventSource", b.EventSource != nil},
		{"Journal", b.Journal != nil},
		{"SeekableJournal", b.SeekableJournal != nil},
		{"Publisher", b.Publisher != nil},
		{"Subscriber", b.Subscriber != nil},
		{"CommandSink", b.CommandSink != nil},
		{"CommandSource", b.CommandSource != nil},
		{"QuerySink", b.QuerySink != nil},
		{"QuerySource", b.QuerySource != nil},
		{"SnapshotStore", b.SnapshotStore != nil},
		{"CheckpointStore", b.CheckpointStore != nil},
		{"ReadModels", b.ReadModels != nil},
	}

	for _, c := range checks {
		if !c.ok {
			t.Errorf("capability %s not set", c.name)
		}
	}
}

// E2E: save events through the preset, load them back, verify ordering.
func TestNew_E2E_EventSaveLoadRoundtrip(t *testing.T) {
	t.Parallel()

	b, err := stackmemory.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", aggID)

	types := []event.Type{"todo.created", "todo.renamed", "todo.completed"}
	payloads := []any{
		map[string]any{"title": "buy milk"},
		map[string]any{"title": "buy oat milk"},
		map[string]any{"at": "now"},
	}

	events, err := event.NewEvents(aggID, "Todo", 0, types, payloads)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("EventSink.Save: %v", err)
	}

	loaded, err := b.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("EventSource.Load: %v", err)
	}

	if len(loaded) != len(events) {
		t.Fatalf("loaded %d events, want %d", len(loaded), len(events))
	}

	for i, typ := range types {
		if loaded[i].Type() != typ {
			t.Errorf("event[%d] type = %s, want %s", i, loaded[i].Type(), typ)
		}
	}
}

// E2E: read-model roundtrip through the preset via stack.ReadModel.
func TestNew_E2E_ReadModelRoundtrip(t *testing.T) {
	t.Parallel()

	b, err := stackmemory.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	store, err := stack.ReadModel[todoView, todoKey](
		b, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[todoView, todoKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel: %v", err)
	}

	ctx := context.Background()

	if err := store.Set(ctx, "1", &todoView{Title: "test", Done: false}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != "test" || got.Done {
		t.Fatalf("read model roundtrip mismatch: %+v", got)
	}
}

func TestNew_CloseReleasesAllResources(t *testing.T) {
	t.Parallel()

	b, err := stackmemory.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
