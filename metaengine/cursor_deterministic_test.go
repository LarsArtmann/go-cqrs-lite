package metaengine_test

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestCursorEncode_MapValueIsDeterministic pins the deterministic encoding of
// map-valued cursors: equal values must always produce identical bytes so a
// cursor can be used as a cache key or ETag component.
func TestCursorEncode_MapValueIsDeterministic(t *testing.T) {
	t.Parallel()

	value := make(map[string]int, 64)
	for i := range 64 {
		value[fmt.Sprintf("key-%02d", i)] = i
	}

	cursor := metaengine.Cursor{Value: value}

	first, err := cursor.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	for range 8 {
		next, err := cursor.Encode()
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		if next != first {
			t.Fatalf("Encode is not deterministic:\n first %q\n next  %q", first, next)
		}
	}

	parsed, err := metaengine.ParseCursor(first)
	if err != nil {
		t.Fatalf("ParseCursor: %v", err)
	}

	decoded, ok := parsed.Value.(map[string]any)
	if !ok {
		t.Fatalf("ParseCursor value = %T, want map[string]any", parsed.Value)
	}

	if len(decoded) != len(value) {
		t.Fatalf("decoded %d keys, want %d", len(decoded), len(value))
	}
}
