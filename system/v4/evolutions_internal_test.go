package system

import (
	"strings"
	"testing"
)

type reifyTarget struct {
	Title string
}

// TestReifyTo_ReturnsErrorOnSchemaMismatch pins the error contract (idea 251):
// a stored value that cannot fit the result type returns an error instead of
// panicking — the fold layer routes it through poison/DLQ, so a schema drift
// must arrive there as a structured error, not as a crash mid-replay.
func TestReifyTo_ReturnsErrorOnSchemaMismatch(t *testing.T) {
	t.Parallel()

	dst := &reifyTarget{}
	src := map[string]any{"Title": 123} // number where a string field is expected

	err := reifyTo(src, dst)
	if err == nil {
		t.Fatal("expected schema-mismatch error, got nil")
	}

	if !strings.Contains(err.Error(), "reifyTo") {
		t.Fatalf("error should name reifyTo, got: %v", err)
	}

	if dst.Title != "" {
		t.Fatalf("target must stay zero-valued on failure, got %+v", *dst)
	}
}

// TestReifyTo_DirectAssignAndJSONRoundTrip covers both healthy shapes: the
// memory-engine fast path (already-typed value, direct assignment) and the
// SQL-engine shape (map[string]any via JSON round-trip), plus the nil no-op.
func TestReifyTo_DirectAssignAndJSONRoundTrip(t *testing.T) {
	t.Parallel()

	typed := &reifyTarget{}
	if err := reifyTo(reifyTarget{Title: "direct"}, typed); err != nil {
		t.Fatalf("direct assign: %v", err)
	}

	if typed.Title != "direct" {
		t.Fatalf("direct assign: want Title %q, got %+v", "direct", *typed)
	}

	fromMap := &reifyTarget{}
	if err := reifyTo(map[string]any{"Title": "from-map"}, fromMap); err != nil {
		t.Fatalf("JSON round-trip: %v", err)
	}

	if fromMap.Title != "from-map" {
		t.Fatalf("JSON round-trip: want Title %q, got %+v", "from-map", *fromMap)
	}

	if err := reifyTo(nil, fromMap); err != nil {
		t.Fatalf("nil source must be a no-op, got %v", err)
	}
}
