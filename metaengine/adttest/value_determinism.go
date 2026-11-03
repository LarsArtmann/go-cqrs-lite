package adttest

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertDeterministicMapEncoding pins the stored-bytes determinism contract on
// one engine instance: the SAME map-typed value written twice must persist as
// byte-identical JSON. encoding/json/v2 dropped v1's implicit map-key sorting,
// so a plain Marshal made every write of the same logical value a different
// byte sequence (the 2026-10-25 finding) — breaking byte-level pins, split
// reads, and differential engine comparisons. Engines that cannot expose raw
// stored bytes (no RawValueReader) skip: their encoders are pinned directly
// (keycodec determinism tests cover the LSM family).
func AssertDeterministicMapEncoding(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Skipf("engine does not implement metaengine.MapBackend")
	}

	raw, ok := eng.(metaengine.RawValueReader)
	if !ok {
		t.Skipf(
			"engine does not implement RawValueReader — encoder-level determinism pins cover it",
		)
	}

	col := fmt.Sprintf("valdet_%d", time.Now().UnixNano())
	ctx := context.Background()

	value := map[string]any{
		"alpha":   1,
		"bravo":   "two",
		"charlie": []any{3, true, nil},
		"delta":   map[string]int{"x": 1, "y": 2},
		"echo":    5.5,
		"foxtrot": "six",
		"golf":    7,
		"hotel":   map[string]any{"inner": "v"},
	}

	twin := map[string]any{
		"hotel":   map[string]any{"inner": "v"},
		"golf":    7,
		"foxtrot": "six",
		"echo":    5.5,
		"delta":   map[string]int{"y": 2, "x": 1},
		"charlie": []any{3, true, nil},
		"bravo":   "two",
		"alpha":   1,
	}

	for key, v := range map[string]map[string]any{"k1": value, "k2": twin} {
		if err := mb.MapSet(ctx, col, key, v); err != nil {
			t.Fatalf("MapSet %s: %v", key, err)
		}
	}

	raw1, ok1, err := raw.GetRawValue(ctx, col, "k1")
	if err != nil || !ok1 {
		t.Fatalf("GetRawValue k1: ok=%v err=%v", ok1, err)
	}

	raw2, ok2, err := raw.GetRawValue(ctx, col, "k2")
	if err != nil || !ok2 {
		t.Fatalf("GetRawValue k2: ok=%v err=%v", ok2, err)
	}

	if !bytes.Equal(raw1, raw2) {
		t.Fatalf(
			"same map value persisted as different bytes:\n k1: %s\n k2: %s\n(map-key sorting lost? json.Deterministic(true) missing on the write path)",
			raw1,
			raw2,
		)
	}

	// Re-writing the SAME key must not change the stored bytes either.
	if err := mb.MapSet(ctx, col, "k1", value); err != nil {
		t.Fatalf("MapSet k1 rewrite: %v", err)
	}

	again, okA, err := raw.GetRawValue(ctx, col, "k1")
	if err != nil || !okA {
		t.Fatalf("GetRawValue k1 rewrite: ok=%v err=%v", okA, err)
	}

	if !bytes.Equal(raw1, again) {
		t.Fatalf(
			"rewriting the same value changed the stored bytes:\n before: %s\n after:  %s",
			raw1,
			again,
		)
	}
}
