//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestStreamAppend_UnrepresentableValueFailsBeforeWrite pins the batch-encode
// contract: one value that cannot be marshaled rejects the whole append before
// any row is written, instead of leaving a partial stream behind.
func TestStreamAppend_UnrepresentableValueFailsBeforeWrite(t *testing.T) {
	t.Parallel()

	eng := newDuckEngineOrSkip(t)
	t.Cleanup(func() { _ = eng.Close() })

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatalf("%T does not implement StreamLogBackend", eng)
	}

	ctx := context.Background()

	err := backend.StreamAppend(ctx, "encode_error_cgo", "s1", []any{
		map[string]string{"ok": "value"},
		make(chan int),
	})
	if err == nil {
		t.Fatal("StreamAppend with an unrepresentable value = nil error")
	}

	values, readErr := backend.StreamRead(ctx, "encode_error_cgo", "s1")
	if readErr != nil {
		t.Fatalf("StreamRead: %v", readErr)
	}

	if len(values) != 0 {
		t.Fatalf("partial append persisted %d value(s), want 0", len(values))
	}
}
