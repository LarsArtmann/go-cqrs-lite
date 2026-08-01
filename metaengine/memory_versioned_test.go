package metaengine

import (
	"context"
	"errors"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestMemoryEngine_VersionedStorage(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngineWithVersioning()
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

// TestMemoryEngine_VersionedStorage_Property uses rapid to generate random
// sequences of MapSet/MapDelete operations on multiple keys, then verifies
// that MapGetAsOf returns the correct value for every key at every recorded
// timestamp.
func TestMemoryEngine_VersionedStorage_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		eng := NewMemoryEngineWithVersioning()
		defer eng.Close()

		ctx := context.Background()
		mb := eng.(MapBackend)
		vs := eng.(VersionedStorage)

		keys := []string{"a", "b", "c"}

		type stateSnapshot struct {
			ts    time.Time
			state map[string]int64 // key → value (absent = deleted/not-set)
		}

		timeline := make([]stateSnapshot, 0, 25)

		current := map[string]int64{}

		numOps := rapid.IntRange(5, 25).Draw(rt, "numOps")

		for range numOps {
			key := rapid.SampledFrom(keys).Draw(rt, "key")

			_, hasKey := current[key]

			if hasKey && rapid.Bool().Draw(rt, "delete") {
				if err := mb.MapDelete(ctx, "items", key); err != nil {
					rt.Fatalf("MapDelete: %v", err)
				}

				delete(current, key)
			} else {
				val := rapid.Int64Range(0, 1000).Draw(rt, "val")
				if err := mb.MapSet(ctx, "items", key, val); err != nil {
					rt.Fatalf("MapSet: %v", err)
				}

				current[key] = val
			}

			// Capture timestamp AFTER the write so engine_ts <= ts.
			ts := time.Now()

			snap := stateSnapshot{ts: ts, state: make(map[string]int64, len(current))}
			for k, v := range current {
				snap.state[k] = v
			}

			timeline = append(timeline, snap)

			time.Sleep(time.Microsecond) // ensure distinct timestamps
		}

		// Verify: at each timestamp, every key matches the reference model.
		for _, snap := range timeline {
			for _, key := range keys {
				val, err := vs.MapGetAsOf(ctx, "items", key, snap.ts)
				expected, existed := snap.state[key]

				if !existed {
					if !errors.Is(err, ErrNotFound) {
						rt.Fatalf("AsOf(%s, %v): expected ErrNotFound, got val=%v err=%v",
							key, snap.ts, val, err)
					}

					continue
				}

				if err != nil {
					rt.Fatalf("AsOf(%s, %v): unexpected error %v", key, snap.ts, err)
				}

				got, ok := val.(int64)
				if !ok {
					rt.Fatalf("AsOf(%s, %v): expected int64, got %T (%v)", key, snap.ts, val, val)
				}

				if got != expected {
					rt.Fatalf("AsOf(%s, %v): got %d, want %d", key, snap.ts, got, expected)
				}
			}
		}

		// Before the first write: every key should return ErrNotFound.
		if len(timeline) > 0 {
			before := timeline[0].ts.Add(-time.Hour)

			for _, key := range keys {
				_, err := vs.MapGetAsOf(ctx, "items", key, before)
				if !errors.Is(err, ErrNotFound) {
					rt.Fatalf("AsOf(%s, before-first): expected ErrNotFound, got %v", key, err)
				}
			}
		}
	})
}

// TestStore_ExecuteAsOf_Integration verifies the full Plan → Apply → ExecuteAsOf
// pipeline for a Map ADT query with versioning enabled.
func TestStore_ExecuteAsOf_Integration(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngineWithVersioning()
	defer eng.Close()

	type UserID string

	type UserCreated struct {
		ID    UserID
		Name  string
		Email string
	}

	type UserUpdated struct {
		ID   UserID
		Name string
	}

	type UserDeleted struct {
		ID UserID
	}

	type FindUser struct{ ID UserID }

	type UserView struct {
		ID    UserID
		Name  string
		Email string
	}

	store, err := Plan(
		[]Engine{eng},
		Query[FindUser, UserView](
			"users",
			On(UserCreated{}, func(e UserCreated) (UserID, UserView) {
				return e.ID, UserView{
					ID:    e.ID,
					Name:  e.Name,
					Email: e.Email,
				} //nolint:staticcheck // intentional field mapping
			}),
			On(UserUpdated{}, func(e UserUpdated, prev UserView) UserView {
				prev.Name = e.Name
				return prev
			}),
			On(UserDeleted{}, Remove[UserView]()),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	ctx := context.Background()

	// Create user
	if err := store.Apply(ctx, "UserCreated", UserCreated{
		ID: "u1", Name: "Alice", Email: "alice@example.com",
	}); err != nil {
		t.Fatalf("Apply UserCreated: %v", err)
	}

	t1 := time.Now()
	time.Sleep(2 * time.Millisecond)

	// Update name
	if err := store.Apply(ctx, "UserUpdated", UserUpdated{
		ID: "u1", Name: "Alice Smith",
	}); err != nil {
		t.Fatalf("Apply UserUpdated: %v", err)
	}

	t2 := time.Now()
	time.Sleep(2 * time.Millisecond)

	// Delete user
	if err := store.Apply(ctx, "UserDeleted", UserDeleted{ID: "u1"}); err != nil {
		t.Fatalf("Apply UserDeleted: %v", err)
	}

	t3 := time.Now()

	// AsOf t1 → original name "Alice"
	val, err := store.ExecuteAsOf(ctx, "users", "u1", t1)
	if err != nil {
		t.Fatalf("ExecuteAsOf(t1): %v", err)
	}

	view, ok := val.(UserView)
	if !ok {
		t.Fatalf("ExecuteAsOf(t1): expected UserView, got %T", val)
	}

	if view.Name != "Alice" {
		t.Errorf("ExecuteAsOf(t1): name = %q, want %q", view.Name, "Alice")
	}

	if view.Email != "alice@example.com" {
		t.Errorf("ExecuteAsOf(t1): email = %q, want %q", view.Email, "alice@example.com")
	}

	// AsOf t2 → updated name "Alice Smith"
	val, err = store.ExecuteAsOf(ctx, "users", "u1", t2)
	if err != nil {
		t.Fatalf("ExecuteAsOf(t2): %v", err)
	}

	view, ok = val.(UserView)
	if !ok {
		t.Fatalf("ExecuteAsOf(t2): expected UserView, got %T", val)
	}

	if view.Name != "Alice Smith" {
		t.Errorf("ExecuteAsOf(t2): name = %q, want %q", view.Name, "Alice Smith")
	}

	// AsOf t3 → ErrNotFound (deleted)
	_, err = store.ExecuteAsOf(ctx, "users", "u1", t3)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ExecuteAsOf(t3): err = %v, want ErrNotFound", err)
	}

	// AsOf before creation → ErrNotFound
	_, err = store.ExecuteAsOf(ctx, "users", "u1", t1.Add(-time.Hour))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ExecuteAsOf(before): err = %v, want ErrNotFound", err)
	}
}
