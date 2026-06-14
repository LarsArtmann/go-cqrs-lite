package encryption

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestEncryptedStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, err := NewAES256GCM(key)
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	inner := eventtest.NewFakeStore()
	store, err := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1,
		[]byte(`{"name":"Alice","email":"alice@example.com"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	ctx := context.Background()
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if string(event.PayloadReadOnly(loaded[0])) != `{"name":"Alice","email":"alice@example.com"}` {
		t.Errorf("payload mismatch: got %s", event.PayloadReadOnly(loaded[0]))
	}
}

func TestEncryptedStore_WithKeyID(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, err := NewEncryptedStore(inner, ed, WithMiddlewareKeyID("key-v1"))
	if err != nil {
		t.Fatal(err)
	}

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"secret":true}`))

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt}, 0)

	raw, err := inner.Load(ctx, ref)
	if err != nil {
		t.Fatalf("raw load: %v", err)
	}

	if len(raw) != 1 {
		t.Fatalf("expected 1 raw event, got %d", len(raw))
	}

	keyID, _ := ExtractKeyID(raw[0])
	if keyID != "key-v1" {
		t.Errorf("key ID: got %q, want %q", keyID, "key-v1")
	}
}

func TestEncryptedStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, err := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt1, _ := event.NewEvent("Created", aggID, "User", 1, []byte(`{"a":1}`))
	evt2, _ := event.NewEvent("Updated", aggID, "User", 2, []byte(`{"a":2}`))

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt1, evt2}, 0)

	loaded, err := store.LoadFromVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event from v1, got %d", len(loaded))
	}

	if string(event.PayloadReadOnly(loaded[0])) != `{"a":2}` {
		t.Errorf("payload: got %s", event.PayloadReadOnly(loaded[0]))
	}
}

func TestEncryptedStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, err := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt1, _ := event.NewEvent("Created", aggID, "User", 1, []byte(`{"x":1}`))
	evt2, _ := event.NewEvent("Updated", aggID, "User", 2, []byte(`{"x":2}`))

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt1, evt2}, 0)

	loaded, err := store.LoadToVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event up to v1, got %d", len(loaded))
	}
}

func TestEncryptedStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, err := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("Created", aggID, "User", 1, []byte(`{}`))

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt}, 0)

	loaded, err := store.LoadToTimestamp(ctx, ref, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestEncryptedStore_AppendBatch(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, err := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("Created", aggID, "User", 1, []byte(`{"batch":true}`))

	ctx := context.Background()
	if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if string(event.PayloadReadOnly(loaded[0])) != `{"batch":true}` {
		t.Errorf("payload: got %s", event.PayloadReadOnly(loaded[0]))
	}
}

func TestEncryptedStore_EmptyPayload(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, _ := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("Tombstoned", aggID, "User", 1, []byte{})

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt}, 0)

	loaded, _ := store.Load(ctx, ref)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if len(event.PayloadReadOnly(loaded[0])) != 0 {
		t.Error("empty payload should pass through unencrypted")
	}
}

func TestEncryptedStore_ReadAll(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, _ := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1,
		[]byte(`{"name":"Alice"}`))

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt}, 0)

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 event, got %d", len(all))
	}

	if string(event.PayloadReadOnly(all[0])) != `{"name":"Alice"}` {
		t.Errorf("payload mismatch: got %s", event.PayloadReadOnly(all[0]))
	}
}

func TestEncryptedStore_ReadFrom(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, _ := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1,
		[]byte(`{"name":"Bob"}`))

	ctx := context.Background()
	_ = store.Save(ctx, ref, []event.Event{evt}, 0)

	fromResult, err := store.ReadFrom(ctx, id.NewEventID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(fromResult) != 1 {
		t.Fatalf("expected 1 event, got %d", len(fromResult))
	}

	if string(event.PayloadReadOnly(fromResult[0])) != `{"name":"Bob"}` {
		t.Errorf("payload mismatch: got %s", event.PayloadReadOnly(fromResult[0]))
	}
}

func TestEncryptedStore_LoadBackwards_NotSupported(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	ed, _ := NewAES256GCM(key)

	inner := eventtest.NewFakeStore()
	store, _ := NewEncryptedStore(inner, ed)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	_, err := store.LoadBackwards(context.Background(), ref)
	if err == nil {
		t.Fatal("expected error for LoadBackwards on non-BackwardsSource store")
	}
}
