package metaengine

import (
	"context"
	"testing"
)

// prefetchFixture is a store over a HELD memory engine plus a prefetch cache, so
// a test can prove a page came from the cache by emptying the engine: if the
// scan still returns rows, they cannot have come from the engine.
type prefetchFixture struct {
	store *Store
	eng   Engine
	cache *PrefetchCache
}

func newPrefetchFixture(t *testing.T, n int) prefetchFixture {
	t.Helper()

	eng := NewMemoryEngine()

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	events := make([]EventInput, 0, n)

	for i := range n {
		id := testTaskID(string(rune('a' + i)))
		events = append(events, EventInput{
			Type:    "task_created",
			Payload: testTask{ID: id, Title: "T" + string(id)},
		})
	}

	if err := store.ApplyBatch(ctx, events); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	return prefetchFixture{store: store, eng: eng, cache: NewPrefetchCache()}
}

// emptyEngine deletes every seeded row, so any subsequent row a scan returns
// must have been served from the PrefetchCache.
func (f prefetchFixture) emptyEngine(t *testing.T, n int) {
	t.Helper()

	mb, ok := f.eng.(MapBackend)
	if !ok {
		t.Fatal("memory engine does not implement MapBackend")
	}

	ctx := context.Background()

	for i := range n {
		id := testTaskID(string(rune('a' + i)))
		if err := mb.MapDelete(ctx, "tasks", id); err != nil {
			t.Fatalf("MapDelete %q: %v", id, err)
		}
	}
}

// TestScanOption_WithPrefetch covers issue #115: PrefetchCache existed but no
// scan option could attach it. The option caches the overflow rows of a page and
// the following call — same shape, same cursor — is served from them.
func TestScanOption_WithPrefetch(t *testing.T) {
	t.Parallel()

	f := newPrefetchFixture(t, 6)
	ctx := context.Background()
	reader := NewReader[testTask](f.store, "tasks")

	page1, cur1, err := reader.ScanPage(ctx,
		WithPrefetch(f.cache), WithSort("Title", false), WithLimit(2))
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}

	if len(page1) != 2 {
		t.Fatalf("page 1 len = %d, want 2", len(page1))
	}

	if cur1 == nil {
		t.Fatal("page 1 cursor is nil, expected another page")
	}

	if got := f.cache.Len(); got != 1 {
		t.Fatalf("cache holds %d pages after page 1, want 1", got)
	}

	// Prove the next page comes from the cache, not the engine.
	f.emptyEngine(t, 6)

	page2, _, err := reader.ScanPage(ctx,
		WithPrefetch(f.cache), WithSort("Title", false), WithLimit(2),
		WithCursor(cur1.Value))
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	if len(page2) != 2 {
		t.Fatalf("page 2 len = %d, want 2 served from cache (engine is empty)", len(page2))
	}

	for i := range page1 {
		if page2[i] == page1[i] {
			t.Errorf("page 2 row %d repeats page 1: %+v", i, page2[i])
		}
	}
}

// TestScanOption_WithPrefetchShapeMismatchMisses pins the shape half of the key:
// a page cached for one filter/sort must never be served to a scan of another
// shape that happens to share the cursor value.
func TestScanOption_WithPrefetchShapeMismatchMisses(t *testing.T) {
	t.Parallel()

	f := newPrefetchFixture(t, 6)
	ctx := context.Background()
	reader := NewReader[testTask](f.store, "tasks")

	_, cur1, err := reader.ScanPage(ctx,
		WithPrefetch(f.cache), WithSort("Title", false), WithLimit(2))
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}

	if cur1 == nil {
		t.Fatal("page 1 cursor is nil, expected another page")
	}

	f.emptyEngine(t, 6)

	// Same cursor, same cache, DIFFERENT sort: the engine answers (nothing left),
	// so the cached rows must not leak into this result.
	rows, _, err := reader.ScanPage(ctx,
		WithPrefetch(f.cache), WithSort("ID", false), WithLimit(2),
		WithCursor(cur1.Value))
	if err != nil {
		t.Fatalf("mismatched scan: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("shape-mismatched scan returned %d cached rows, want 0", len(rows))
	}
}

// TestScanOption_WithPrefetchNilDisables pins the precedence rule: an explicit
// nil option turns caching off for that call even when the reader carries a
// default cache, so one reader can serve both cached and uncached scans.
func TestScanOption_WithPrefetchNilDisables(t *testing.T) {
	t.Parallel()

	f := newPrefetchFixture(t, 6)
	ctx := context.Background()
	reader := NewReader[testTask](f.store, "tasks").WithPrefetch(f.cache)

	if _, _, err := reader.ScanPage(ctx,
		WithPrefetch(nil), WithSort("Title", false), WithLimit(2)); err != nil {
		t.Fatalf("ScanPage: %v", err)
	}

	if got := f.cache.Len(); got != 0 {
		t.Fatalf("explicit WithPrefetch(nil) still cached %d pages", got)
	}
}

// TestScanOption_WithPrefetchSharedAcrossReaders pins that the CACHE is the unit
// of reuse, not the reader — the HTTP-handler shape, where a fresh reader is
// built per request and only the cache survives.
func TestScanOption_WithPrefetchSharedAcrossReaders(t *testing.T) {
	t.Parallel()

	f := newPrefetchFixture(t, 6)
	ctx := context.Background()

	_, cur1, err := NewReader[testTask](f.store, "tasks").ScanPage(ctx,
		WithPrefetch(f.cache), WithSort("Title", false), WithLimit(2))
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}

	if cur1 == nil {
		t.Fatal("page 1 cursor is nil, expected another page")
	}

	f.emptyEngine(t, 6)

	page2, _, err := NewReader[testTask](f.store, "tasks").ScanPage(ctx,
		WithPrefetch(f.cache), WithSort("Title", false), WithLimit(2),
		WithCursor(cur1.Value))
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	if len(page2) != 2 {
		t.Fatalf("page 2 len = %d, want 2 served from the shared cache", len(page2))
	}
}
