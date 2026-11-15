package adttest

import (
	"context"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertMapKeyIdentity pins the KEYS-class determinism contract on one engine
// instance: a map-typed key must resolve to ONE storage key regardless of the
// map's iteration order at encode time. encoding/json/v2 dropped v1's implicit
// map-key sorting, so a plain Marshal in a key encoder made the SAME logical
// key land on DIFFERENT storage keys (the 2026-10-25 finding): a Get missed
// the row a Set wrote (split read), a second Set added a second row (broken
// uniqueness), and a Delete left the original row behind (orphan). The
// value-bytes twin is [AssertDeterministicMapEncoding]; this is the strictly
// more dangerous keys half — key nondeterminism corrupts state, value
// nondeterminism only breaks byte identity.
func AssertMapKeyIdentity(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	mb := requireMapBackend(t, eng)

	col := fmt.Sprintf("keyid_%d", time.Now().UnixNano())
	ctx := context.Background()

	keyA := map[string]any{"alpha": 1, "bravo": "two", "charlie": map[string]int{"x": 1, "y": 2}}
	keyB := map[string]any{
		"charlie": map[string]int{"y": 2, "x": 1},
		"bravo":   "two",
		"alpha":   1,
	}

	if err := mb.MapSet(ctx, col, keyA, "first"); err != nil {
		t.Fatalf("MapSet keyA: %v", err)
	}

	got, ok, err := mb.MapGet(ctx, col, keyB)
	if err != nil {
		t.Fatalf("MapGet keyB: %v", err)
	}

	if !ok {
		t.Fatal("MapGet with an equal-but-differently-built map key missed the row a MapSet wrote" +
			" (split read: map-key encoding is not deterministic)")
	}

	if got != "first" {
		t.Fatalf("MapGet keyB = %v, want \"first\"", got)
	}

	if err := mb.MapSet(ctx, col, keyB, "second"); err != nil {
		t.Fatalf("MapSet keyB: %v", err)
	}

	if got, ok, err := mb.MapGet(ctx, col, keyA); err != nil || !ok {
		t.Fatalf("MapGet keyA after overwrite: ok=%v err=%v", ok, err)
	} else if got != "second" {
		t.Fatalf("overwrite with an equal key did not replace the row: got %v, want \"second\""+
			" (broken uniqueness: two storage rows for one logical key)", got)
	}

	if err := mb.MapDelete(ctx, col, keyB); err != nil {
		t.Fatalf("MapDelete keyB: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, col, keyA); err != nil {
		t.Fatalf("MapGet keyA after delete: %v", err)
	} else if ok {
		t.Fatal("MapGet hit a row after MapDelete with an equal key" +
			" (orphaned row: delete and set resolved different storage keys)")
	}
}
