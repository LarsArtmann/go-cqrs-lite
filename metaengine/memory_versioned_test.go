package metaengine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryEngine_VersionedStorage(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(MapBackend)

	// Write v1
	if err := mb.MapSet(
		ctx,
		"users",
		"u1",
		map[string]any{"name": "Alice", "v": float64(1)},
	); err != nil {
		t.Fatal(err)
	}

	t1 := time.Now()
	time.Sleep(1 * time.Millisecond)

	// Write v2
	if err := mb.MapSet(
		ctx,
		"users",
		"u1",
		map[string]any{"name": "Alice", "v": float64(2)},
	); err != nil {
		t.Fatal(err)
	}

	t2 := time.Now()
	time.Sleep(1 * time.Millisecond)

	// Delete
	if err := mb.MapDelete(ctx, "users", "u1"); err != nil {
		t.Fatal(err)
	}

	t3 := time.Now()

	vs := eng.(VersionedStorage)

	// AsOf t1 → should return v1
	val, err := vs.MapGetAsOf(ctx, "users", "u1", t1)
	if err != nil {
		t.Fatalf("MapGetAsOf(t1): %v", err)
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("MapGetAsOf(t1): expected map, got %T", val)
	}

	if m["v"] != float64(1) {
		t.Errorf("MapGetAsOf(t1): v = %v, want 1", m["v"])
	}

	// AsOf t2 → should return v2
	val, err = vs.MapGetAsOf(ctx, "users", "u1", t2)
	if err != nil {
		t.Fatalf("MapGetAsOf(t2): %v", err)
	}

	m, ok = val.(map[string]any)
	if !ok {
		t.Fatalf("MapGetAsOf(t2): expected map, got %T", val)
	}

	if m["v"] != float64(2) {
		t.Errorf("MapGetAsOf(t2): v = %v, want 2", m["v"])
	}

	// AsOf t3 → should return ErrNotFound (deleted)
	_, err = vs.MapGetAsOf(ctx, "users", "u1", t3)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("MapGetAsOf(t3): err = %v, want ErrNotFound", err)
	}

	// MapExistsAsOf checks
	exists, _ := vs.MapExistsAsOf(ctx, "users", "u1", t1)
	if !exists {
		t.Error("MapExistsAsOf(t1): expected true")
	}

	exists, _ = vs.MapExistsAsOf(ctx, "users", "u1", t3)
	if exists {
		t.Error("MapExistsAsOf(t3): expected false (deleted)")
	}

	// Non-existent key
	exists, _ = vs.MapExistsAsOf(ctx, "users", "nonexistent", t1)
	if exists {
		t.Error("MapExistsAsOf(nonexistent): expected false")
	}
}
