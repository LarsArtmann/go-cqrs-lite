package system_test

import (
	"context"
	"maps"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestCommandAdapter_MetadataRoundTrip pins that tracing + custom metadata
// survive the serialization envelope path (encode to TEXT, decode back) that
// persistent StreamLogBackends use. Guards the metadata guarantees documented
// on encodeCommand: full-fidelity for marshalable metadata, never partial JSON.
func TestCommandAdapter_MetadataRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("memory engine must implement StreamLogBackend")
	}

	adapter := system.NewCommandAdapter(backend, "commands", system.WithCommandSerialization())

	ref := command.NewStreamRef("CmdTask", id.NewStreamID())

	want := command.Metadata{
		Tracing: metadata.Tracing{
			CorrelationID: id.NewCorrelationID(),
			CausationID:   id.NewCausationID(),
			UserID:        id.NewUserID(),
			RequestID:     id.NewRequestID(),
		},
		Custom: map[command.MetadataKey]string{
			"tenant": "acme",
			"source": "test",
		},
	}

	cmd, err := command.NewPersistedCommand(
		"task.create", ref, []byte(`{"title":"a"}`),
		command.WithCommandMetadata(want),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := adapter.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := adapter.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Load returned %d commands, want 1", len(loaded))
	}

	got := loaded[0].Metadata()
	if got.Tracing != want.Tracing {
		t.Errorf("tracing mismatch:\n got: %+v\nwant: %+v", got.Tracing, want.Tracing)
	}
	if !maps.Equal(got.Custom, want.Custom) {
		t.Errorf("custom metadata mismatch:\n got: %v\nwant: %v", got.Custom, want.Custom)
	}
}

// TestQueryAdapter_MetadataRoundTrip is the query-side twin of the command
// metadata roundtrip pin (encodeQuery/decodeQuery envelope path).
func TestQueryAdapter_MetadataRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("memory engine must implement StreamLogBackend")
	}

	adapter := system.NewQueryAdapter(backend, "queries", system.WithQuerySerialization())

	want := query.Metadata{
		Tracing: metadata.Tracing{
			CorrelationID: id.NewCorrelationID(),
			RequestID:     id.NewRequestID(),
		},
		Custom: map[query.MetadataKey]string{
			"view": "tasks",
		},
	}

	q, err := query.NewPersistedQuery(
		"tasks.list", []byte(`{"limit":10}`),
		query.WithQueryMetadata(want),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	if err := adapter.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	loaded, err := adapter.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("ReadAllQueries returned %d queries, want 1", len(loaded))
	}

	got := loaded[0].Metadata()
	if got.Tracing != want.Tracing {
		t.Errorf("tracing mismatch:\n got: %+v\nwant: %+v", got.Tracing, want.Tracing)
	}
	if !maps.Equal(got.Custom, want.Custom) {
		t.Errorf("custom metadata mismatch:\n got: %v\nwant: %v", got.Custom, want.Custom)
	}
}
