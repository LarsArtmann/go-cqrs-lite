package mysqlengine

import (
	"context"
	"testing"
)

// TestStreamAppend_UnrepresentableValueFailsBeforeWrite pins the batch-encode
// contract: a value that cannot be marshaled rejects the append before the
// transaction opens, so no partial stream is committed.
//
// Runs only when MYSQL_TEST_DSN is set.
func TestStreamAppend_UnrepresentableValueFailsBeforeWrite(t *testing.T) {
	eng := newInternalEngine(t)
	t.Cleanup(func() { _ = eng.Close() })

	ctx := context.Background()
	collection := "encode_error_" + runSuffix

	err := eng.StreamAppend(ctx, collection, "s1", []any{
		map[string]string{"ok": "value"},
		make(chan int),
	})
	if err == nil {
		t.Fatal("StreamAppend with an unrepresentable value = nil error")
	}

	values, readErr := eng.StreamRead(ctx, collection, "s1")
	if readErr != nil {
		t.Fatalf("StreamRead: %v", readErr)
	}

	if len(values) != 0 {
		t.Fatalf("partial append persisted %d value(s), want 0", len(values))
	}
}
