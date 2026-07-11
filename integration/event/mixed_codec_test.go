package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestMixedCodecStream proves JSON-encoded and CBOR-encoded events coexist
// in the same event store and decode correctly with their respective codecs.
// This validates the library's dual-codec guarantee: both JSON and CBOR are
// first-class, and mixed streams work transparently.
func TestMixedCodecStream(t *testing.T) {
	t.Parallel()

	type UserCreated struct {
		Name string
	}

	type UserUpdated struct {
		Name string
	}

	store := memory.NewMemoryStore()
	ctx := t.Context()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("User", aggID)

	// Event 1: JSON-encoded (explicit — DefaultCodec is CBOR now)
	evt1, err := event.New(
		"user.created", aggID, "User", 1,
		UserCreated{Name: "Alice"},
		event.WithCodec(codec.JSONCodec{}),
	)
	if err != nil {
		t.Fatalf("New JSON event: %v", err)
	}

	if evt1.Encoding() != codec.EncodingJSON {
		t.Fatalf("evt1 encoding = %s, want json", evt1.Encoding())
	}

	// Event 2: CBOR-encoded
	evt2, err := event.New(
		"user.updated", aggID, "User", 2,
		UserUpdated{Name: "Bob"},
		event.WithCodec(codec.CBORCodec{}),
	)
	if err != nil {
		t.Fatalf("New CBOR event: %v", err)
	}

	if evt2.Encoding() != codec.EncodingCBOR {
		t.Fatalf("evt2 encoding = %s, want cbor", evt2.Encoding())
	}

	// Save both to the same store
	err = store.Save(ctx, ref, []event.Event{evt1, evt2}, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load back
	events, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Event 1: decode with JSONCodec (encoding stamps must survive roundtrip)
	if events[0].Encoding() != codec.EncodingJSON {
		t.Fatalf("loaded evt1 encoding = %s, want json", events[0].Encoding())
	}

	created, err := event.DecodePayload[UserCreated](events[0], codec.JSONCodec{})
	if err != nil {
		t.Fatalf("Decode JSON event: %v", err)
	}

	if created.Name != "Alice" {
		t.Fatalf("created.Name = %q, want Alice", created.Name)
	}

	// Event 2: decode with CBORCodec
	if events[1].Encoding() != codec.EncodingCBOR {
		t.Fatalf("loaded evt2 encoding = %s, want cbor", events[1].Encoding())
	}

	updated, err := event.DecodePayload[UserUpdated](events[1], codec.CBORCodec{})
	if err != nil {
		t.Fatalf("Decode CBOR event: %v", err)
	}

	if updated.Name != "Bob" {
		t.Fatalf("updated.Name = %q, want Bob", updated.Name)
	}

	// Cross-decode must now fail (symmetric validation)
	_, err = event.DecodePayload[UserCreated](events[0], codec.CBORCodec{})
	if err == nil {
		t.Fatal("expected rejection when decoding JSON event with CBOR codec")
	}

	_, err = event.DecodePayload[UserUpdated](events[1], codec.JSONCodec{})
	if err == nil {
		t.Fatal("expected rejection when decoding CBOR event with JSON codec")
	}
}
