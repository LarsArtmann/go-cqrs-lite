package commandlifecycle

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystore "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestBoundedMap_EvictsOldestAtCapacity(t *testing.T) {
	m := newBoundedMap[int](2)
	m.put("a", 1)
	m.put("b", 2)
	m.put("c", 3)

	if _, ok := m.get("a"); ok {
		t.Fatal("expected oldest entry 'a' to be evicted at capacity 2")
	}

	if v, ok := m.get("c"); !ok || v != 3 {
		t.Fatalf("expected 'c'=3 to survive, got %d ok=%v", v, ok)
	}

	if m.len() != 2 {
		t.Fatalf("expected len 2, got %d", m.len())
	}
}

func TestBoundedMap_UpdateDoesNotEvict(t *testing.T) {
	m := newBoundedMap[int](2)
	m.put("a", 1)
	m.put("b", 2)
	m.put("a", 10)

	if m.len() != 2 {
		t.Fatalf("update at capacity must not evict, got len %d", m.len())
	}
}

func TestBoundedMap_UnboundedWhenCapacityNonPositive(t *testing.T) {
	m := newBoundedMap[int](0)
	for i := range 100 {
		m.put(string(rune('a'+i%26)), i)
	}

	if m.len() == 0 {
		t.Fatal("expected unbounded map to keep entries")
	}
}

func TestBoundedMap_DeleteLeavesLazyReclaim(t *testing.T) {
	m := newBoundedMap[int](2)
	m.put("a", 1)
	m.put("b", 2)
	m.delete("a")
	m.put("c", 3)
	m.put("d", 4)

	if m.len() > 2 {
		t.Fatalf("entries must stay bounded, got %d", m.len())
	}
}

func TestRecorder_VersionCacheCapacity_ReSeedsAfterEviction(t *testing.T) {
	store := memorystore.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	recorder := NewRecorder(store, WithVersionCacheCapacity(1))
	ctx := context.Background()

	cmds := make([]*command.BasicCommand, 3)
	for i := range cmds {
		cmd, err := command.New("create_user", id.NewStreamID())
		if err != nil {
			t.Fatalf("command.New: %v", err)
		}

		cmds[i] = cmd
	}

	for _, cmd := range cmds {
		if err := recorder.RecordReceived(ctx, cmd); err != nil {
			t.Fatalf("RecordReceived(%s): %v", cmd.ID(), err)
		}
	}

	if recorder.versions.len() > 1 {
		t.Fatalf("version cache exceeded capacity: %d entries", recorder.versions.len())
	}

	for _, cmd := range cmds {
		if err := recorder.RecordCompleted(ctx, cmd); err != nil {
			t.Fatalf("RecordCompleted after eviction(%s): %v", cmd.ID(), err)
		}
	}

	for i, cmd := range cmds {
		events, err := store.Load(ctx, LifecycleStreamRef(cmd))
		if err != nil {
			t.Fatalf("Load(%s): %v", cmd.ID(), err)
		}

		if len(events) != 2 {
			t.Fatalf("cmd %d: expected received+completed, got %d events", i, len(events))
		}

		if got := events[1].Version(); got != 2 {
			t.Fatalf("cmd %d: completed event version = %d, want 2", i, got)
		}
	}
}

func TestRecorder_UnboundedWhenCapacityZero(t *testing.T) {
	store := memorystore.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	recorder := NewRecorder(store, WithVersionCacheCapacity(0))
	ctx := context.Background()

	for range 5 {
		cmd, err := command.New("create_user", id.NewStreamID())
		if err != nil {
			t.Fatalf("command.New: %v", err)
		}

		if err := recorder.RecordReceived(ctx, cmd); err != nil {
			t.Fatalf("RecordReceived: %v", err)
		}
	}

	if recorder.versions.len() != 5 {
		t.Fatalf("unbounded cache should keep all entries, got %d", recorder.versions.len())
	}
}
