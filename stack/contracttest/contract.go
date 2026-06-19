// Package contracttest provides a reusable contract test suite for
// [stack.Bundle] presets.
//
// Every preset (memory, sqlite, pebble, postgres) must satisfy the same
// behavioral contract: Bundle fields populated, event roundtrip, read-model
// roundtrip, and idempotent Close. RunSuite verifies all of these against a
// caller-provided factory so a new preset gets the full test suite for free.
//
//	func TestContract(t *testing.T) {
//	    contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
//	        return sqlite.New(filepath.Join(t.TempDir(), "test.db"))
//	    })
//	}
package contracttest

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
)

// Factory creates a fresh [stack.Bundle] for testing. Each call should produce
// an independent Bundle with its own backing resources (temp directory, etc.).
// The caller is responsible for calling Bundle.Close.
type Factory func(t *testing.T) (*stack.Bundle, error)

type contractKey string

func (k contractKey) String() string { return string(k) }

type contractView struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// RunSuite runs the full contract test suite against the given factory.
// Each subtest gets its own fresh Bundle via the factory.
func RunSuite(t *testing.T, factory Factory) {
	t.Helper()

	t.Run("BundleFields", func(t *testing.T) { testBundleFields(t, factory) })
	t.Run("EventRoundtrip", func(t *testing.T) { testEventRoundtrip(t, factory) })
	t.Run("CommandRoundtrip", func(t *testing.T) { testCommandRoundtrip(t, factory) })
	t.Run("ReadModelRoundtrip", func(t *testing.T) { testReadModelRoundtrip(t, factory) })
	t.Run("CloseIdempotent", func(t *testing.T) { testCloseIdempotent(t, factory) })
}

func testBundleFields(t *testing.T, factory Factory) {
	t.Parallel()

	b, err := factory(t)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Error("EventSink is nil")
	}

	if b.EventSource == nil {
		t.Error("EventSource is nil")
	}

	if b.Publisher == nil {
		t.Error("Publisher is nil")
	}

	if b.CommandSink == nil {
		t.Error("CommandSink is nil")
	}

	if b.SnapshotStore == nil {
		t.Error("SnapshotStore is nil")
	}

	if b.ReadModels == nil {
		t.Error("ReadModels is nil")
	}
}

func testEventRoundtrip(t *testing.T, factory Factory) {
	t.Parallel()

	b, err := factory(t)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Contract", aggID)

	events, err := event.NewEvents(
		aggID, "Contract", 0,
		[]event.Type{"contract.created", "contract.updated"},
		[]any{
			map[string]any{"name": "alpha"},
			map[string]any{"name": "beta"},
		},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := b.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	if loaded[0].Type() != "contract.created" {
		t.Errorf("event[0] type = %s, want contract.created", loaded[0].Type())
	}

	if loaded[1].Type() != "contract.updated" {
		t.Errorf("event[1] type = %s, want contract.updated", loaded[1].Type())
	}
}

func testCommandRoundtrip(t *testing.T, factory Factory) {
	t.Parallel()

	b, err := factory(t)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.CommandSink == nil {
		t.Skip("CommandSink not available")
	}

	ctx := context.Background()
	ref := command.NewAggregateRef("Contract", id.NewAggregateID())

	cmd, err := command.NewPersistedCommand(
		"contract.create", ref, []byte(`{"action":"test"}`),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := b.CommandSink.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save command: %v", err)
	}

	loaded, err := b.CommandSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load commands: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}
}

func testReadModelRoundtrip(t *testing.T, factory Factory) {
	t.Parallel()

	b, err := factory(t)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	defer func() { _ = b.Close() }()

	store, err := stack.ReadModel[contractView, contractKey](
		b, codec.JSONCodec{},
		readmodel.WithKeyPrefix[contractView, contractKey]("contract:"),
	)
	if err != nil {
		t.Fatalf("ReadModel: %v", err)
	}

	ctx := context.Background()

	if err := store.Set(ctx, "1", &contractView{Title: "test", Done: false}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != "test" || got.Done {
		t.Fatalf("read model mismatch: %+v", got)
	}
}

func testCloseIdempotent(t *testing.T, factory Factory) {
	t.Parallel()

	b, err := factory(t)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
