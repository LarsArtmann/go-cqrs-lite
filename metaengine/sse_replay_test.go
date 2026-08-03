package metaengine

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// --- SSEReplay unit tests ---

func TestSSEReplay_RecordAndReplay(t *testing.T) {
	t.Parallel()

	r := NewSSEReplay[testTask](10)

	// No entries initially.
	if entries := r.Replay(0); entries != nil {
		t.Fatalf("expected nil for empty replay, got %d entries", len(entries))
	}

	// Record three values.
	seq1 := r.record(testTask{ID: "t1", Title: "Task 1"})
	seq2 := r.record(testTask{ID: "t2", Title: "Task 2"})
	seq3 := r.record(testTask{ID: "t3", Title: "Task 3"})

	if seq1 != 1 || seq2 != 2 || seq3 != 3 {
		t.Fatalf("expected seqs 1,2,3 got %d,%d,%d", seq1, seq2, seq3)
	}

	if latest := r.LatestSeq(); latest != 3 {
		t.Fatalf("expected latest seq 3, got %d", latest)
	}

	// Replay all from seq 0.
	all := r.Replay(0)
	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}

	if all[0].Seq != 1 || all[0].Value.ID != "t1" {
		t.Errorf("entry 0: expected seq=1 id=t1, got seq=%d id=%s", all[0].Seq, all[0].Value.ID)
	}

	if all[2].Seq != 3 || all[2].Value.ID != "t3" {
		t.Errorf("entry 2: expected seq=3 id=t3, got seq=%d id=%s", all[2].Seq, all[2].Value.ID)
	}

	// Replay from seq 2 (should get only seq 3).
	after2 := r.Replay(2)
	if len(after2) != 1 {
		t.Fatalf("expected 1 entry after seq 2, got %d", len(after2))
	}

	if after2[0].Seq != 3 {
		t.Errorf("expected seq=3, got seq=%d", after2[0].Seq)
	}

	// Replay from latest seq (should get nothing).
	none := r.Replay(3)
	if none != nil {
		t.Fatalf("expected nil for replay after latest, got %d entries", len(none))
	}
}

func TestSSEReplay_RingBufferEviction(t *testing.T) {
	t.Parallel()

	r := NewSSEReplay[testTask](3) // capacity 3

	r.record(testTask{ID: "t1"})
	r.record(testTask{ID: "t2"})
	r.record(testTask{ID: "t3"})
	r.record(testTask{ID: "t4"}) // should evict t1

	all := r.Replay(0)
	if len(all) != 3 {
		t.Fatalf("expected 3 entries (capacity), got %d", len(all))
	}

	if all[0].Value.ID != "t2" {
		t.Errorf("expected oldest surviving entry t2, got %s", all[0].Value.ID)
	}

	if all[2].Value.ID != "t4" {
		t.Errorf("expected newest entry t4, got %s", all[2].Value.ID)
	}

	// seq should still be monotonic despite eviction.
	if all[0].Seq != 2 {
		t.Errorf("expected oldest seq=2, got %d", all[0].Seq)
	}
}

func TestSSEReplay_DefaultCapacity(t *testing.T) {
	t.Parallel()

	r := NewSSEReplay[testTask](0) // 0 → default 64

	for i := range 70 {
		r.record(testTask{ID: testTaskID("t" + string(rune('a'+i%26)))})
	}

	all := r.Replay(0)
	if len(all) != 64 {
		t.Fatalf("expected 64 entries (default cap), got %d", len(all))
	}
}

// --- Watcher.WithReplay integration tests ---

func TestWatcher_WithReplay_RecordsValues(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	ctx := context.Background()

	// Apply events — watcher should record them in the replay journal.
	_ = store.Apply(ctx, "task_created", testTask{ID: "r1", Title: "Replay 1"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "r2", Title: "Replay 2"})

	// Give the watcher notification time to propagate.
	time.Sleep(100 * time.Millisecond)

	if latest := replay.LatestSeq(); latest != 2 {
		t.Fatalf("expected latest seq 2, got %d", latest)
	}

	entries := replay.Replay(0)
	if len(entries) != 2 {
		t.Fatalf("expected 2 replay entries, got %d", len(entries))
	}

	if entries[0].Value.ID != "r1" {
		t.Errorf("entry 0: expected r1, got %s", entries[0].Value.ID)
	}

	if entries[1].Value.ID != "r2" {
		t.Errorf("entry 1: expected r2, got %s", entries[1].Value.ID)
	}
}

func TestWatcher_WatchWithSeq_ReturnsSeqValues(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seqCh := watcher.WatchWithSeq(ctx, nil)

	_ = store.Apply(ctx, "task_created", testTask{ID: "ws1", Title: "WithSeq"})

	select {
	case sv := <-seqCh:
		if sv.Value.ID != "ws1" {
			t.Errorf("expected ws1, got %s", sv.Value.ID)
		}

		if sv.Seq == 0 {
			t.Error("expected non-zero seq when replay is attached")
		}

		if sv.Seq != replay.LatestSeq() {
			t.Errorf("seq mismatch: got %d, latest %d", sv.Seq, replay.LatestSeq())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for WatchWithSeq value")
	}
}

// --- SSE Last-Event-ID reconnection end-to-end test ---

func TestSSE_LastEventID_Reconnect(t *testing.T) {
	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	watcher.WithReplay(100)
	defer watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(
			w, r, watcher,
			WithSSETimeout(5*time.Second),
		)
	})

	srv := &http.Server{Handler: mux}
	defer srv.Close()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv.Addr = ln.Addr().String()
	go srv.Serve(ln)

	time.Sleep(100 * time.Millisecond)

	// Phase 1: Connect and receive events.
	conn1, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn1.Close()

	_, _ = conn1.Write([]byte("GET /events HTTP/1.0\r\nHost: localhost\r\n\r\n"))
	time.Sleep(200 * time.Millisecond)

	// Apply events while client 1 is connected.
	_ = store.Apply(ctx, "task_created", testTask{ID: "rc1", Title: "Reconnect 1"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "rc2", Title: "Reconnect 2"})

	// Read from client 1 until we get both events and the id fields.
	var data1 string
	deadline := time.Now().Add(5 * time.Second)
	buf := make([]byte, 8192)

	for time.Now().Before(deadline) {
		conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn1.Read(buf)
		if err != nil {
			break
		}

		data1 += string(buf[:n])

		if strings.Contains(data1, "Reconnect 2") {
			break
		}
	}

	// Verify id: field is present.
	if !strings.Contains(data1, "id: 1") {
		t.Errorf("expected 'id: 1' in SSE output, got: %s", data1)
	}

	if !strings.Contains(data1, "id: 2") {
		t.Errorf("expected 'id: 2' in SSE output, got: %s", data1)
	}

	if !strings.Contains(data1, "Reconnect 1") {
		t.Errorf("expected 'Reconnect 1' in data, got: %s", data1)
	}

	// Phase 2: Disconnect client 1, apply more events, reconnect with Last-Event-ID.
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	_ = store.Apply(ctx, "task_created", testTask{ID: "rc3", Title: "Reconnect 3"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "rc4", Title: "Reconnect 4"})

	time.Sleep(100 * time.Millisecond)

	// Reconnect with Last-Event-ID: 2 (last seq received).
	conn2, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn2.Close()

	_, _ = conn2.Write(
		[]byte("GET /events HTTP/1.0\r\nHost: localhost\r\nLast-Event-ID: 2\r\n\r\n"),
	)

	// Read replayed events.
	var data2 string
	deadline2 := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline2) {
		conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn2.Read(buf)
		if err != nil {
			break
		}

		data2 += string(buf[:n])

		if strings.Contains(data2, "Reconnect 3") && strings.Contains(data2, "Reconnect 4") {
			break
		}
	}

	// Should have replayed events 3 and 4 (after seq 2).
	if !strings.Contains(data2, "id: 3") {
		t.Errorf("expected 'id: 3' in replayed output, got: %s", data2)
	}

	if !strings.Contains(data2, "id: 4") {
		t.Errorf("expected 'id: 4' in replayed output, got: %s", data2)
	}

	if !strings.Contains(data2, "Reconnect 3") {
		t.Errorf("expected 'Reconnect 3' in replayed data, got: %s", data2)
	}

	if !strings.Contains(data2, "Reconnect 4") {
		t.Errorf("expected 'Reconnect 4' in replayed data, got: %s", data2)
	}

	// Should NOT have replayed events 1 and 2 (those are before Last-Event-ID).
	if strings.Contains(data2, "Reconnect 1") {
		t.Errorf("replay should not include 'Reconnect 1' (before Last-Event-ID), got: %s", data2)
	}

	if strings.Contains(data2, "Reconnect 2") {
		t.Errorf("replay should not include 'Reconnect 2' (before Last-Event-ID), got: %s", data2)
	}
}

func TestSSE_ReplayLimit(t *testing.T) {
	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	watcher.WithReplay(100)
	defer watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Apply 5 events.
	for i := range 5 {
		_ = store.Apply(ctx, "task_created", testTask{
			ID:    testTaskID("rl" + string(rune('1'+i))),
			Title: "ReplayLimit " + string(rune('1'+i)),
		})
	}

	time.Sleep(200 * time.Millisecond)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(
			w, r, watcher,
			WithSSETimeout(2*time.Second),
			WithSSEReplayLimit(2), // cap replay at 2 events
		)
	})

	srv := &http.Server{Handler: mux}
	defer srv.Close()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv.Addr = ln.Addr().String()
	go srv.Serve(ln)

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	// Connect with Last-Event-ID: 0 (replay all, but capped at 2).
	_, _ = conn.Write([]byte("GET /events HTTP/1.0\r\nHost: localhost\r\nLast-Event-ID: 0\r\n\r\n"))

	var data string
	deadline := time.Now().Add(5 * time.Second)
	buf := make([]byte, 8192)

	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			break
		}

		data += string(buf[:n])

		// Wait for enough data.
		if strings.Count(data, "id:") >= 2 {
			break
		}
	}

	// Should have at most 2 id: lines (replay capped at 2).
	idCount := strings.Count(data, "id:")
	if idCount > 2 {
		t.Errorf("expected at most 2 replayed events, got %d id fields", idCount)
	}
}

// --- SSE reconnect with SQLite engine (reify fallback path) ---

// TestSSE_ReconnectWithSQLite verifies that Last-Event-ID reconnection works
// end-to-end when the backing engine is SQLite. SQLite returns map[string]any
// from JSON-decoded rows, so the replay journal and watcher must reify values
// to the typed V via JSON round-trip. This test catches regressions in the
// reifyWatcherValue fallback path under SSE replay.
func TestSSE_ReconnectWithSQLite(t *testing.T) {
	store := newSQLiteTestStore(t)
	defer store.Close()

	watcher := NewWatcher[testTask](store, "tasks")
	watcher.WithReplay(100)
	defer watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(w, r, watcher, WithSSETimeout(5*time.Second))
	})

	srv := &http.Server{Handler: mux}
	defer srv.Close()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv.Addr = ln.Addr().String()
	go srv.Serve(ln)

	time.Sleep(100 * time.Millisecond)

	// Phase 1: connect, receive live events from SQLite.
	conn1, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn1.Close()

	_, _ = conn1.Write([]byte("GET /events HTTP/1.0\r\nHost: localhost\r\n\r\n"))
	time.Sleep(200 * time.Millisecond)

	_ = store.Apply(ctx, "task_created", testTask{ID: "s1", Title: "SQLite Live 1"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "s2", Title: "SQLite Live 2"})

	var data1 string

	buf := make([]byte, 8192)
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn1.Read(buf)
		if err != nil {
			break
		}

		data1 += string(buf[:n])

		if strings.Contains(data1, "SQLite Live 2") {
			break
		}
	}

	if !strings.Contains(data1, "id: 1") {
		t.Errorf("expected 'id: 1' in SSE output, got: %s", data1)
	}

	if !strings.Contains(data1, "SQLite Live 1") {
		t.Errorf("expected 'SQLite Live 1' in data, got: %s", data1)
	}

	// Phase 2: disconnect, apply more events, reconnect with Last-Event-ID: 2.
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	_ = store.Apply(ctx, "task_created", testTask{ID: "s3", Title: "SQLite Replay 3"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "s4", Title: "SQLite Replay 4"})

	time.Sleep(100 * time.Millisecond)

	conn2, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn2.Close()

	_, _ = conn2.Write(
		[]byte("GET /events HTTP/1.0\r\nHost: localhost\r\nLast-Event-ID: 2\r\n\r\n"),
	)

	var data2 string
	deadline2 := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline2) {
		conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn2.Read(buf)
		if err != nil {
			break
		}

		data2 += string(buf[:n])

		if strings.Contains(data2, "SQLite Replay 3") && strings.Contains(data2, "SQLite Replay 4") {
			break
		}
	}

	if !strings.Contains(data2, "id: 3") {
		t.Errorf("expected 'id: 3' in replayed output, got: %s", data2)
	}

	if !strings.Contains(data2, "id: 4") {
		t.Errorf("expected 'id: 4' in replayed output, got: %s", data2)
	}

	if !strings.Contains(data2, "SQLite Replay 3") {
		t.Errorf("expected 'SQLite Replay 3' in replayed data, got: %s", data2)
	}

	// Should NOT include pre-Last-Event-ID events.
	if strings.Contains(data2, "SQLite Live 1") {
		t.Errorf("replay should not include 'SQLite Live 1' (before Last-Event-ID), got: %s", data2)
	}
}

// --- Cursor/PrefetchCache integration tests ---

func TestPrefetchCache_CursorEncodeRoundTrip(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	for i := range 10 {
		_ = store.Apply(ctx, "task_created", testTask{
			ID:    testTaskID(fmt.Sprintf("c%d", i)),
			Title: fmt.Sprintf("Cursor-%02d", i),
		})
	}

	// Page 1: no cursor.
	page1, cursor1, err := reader.ScanPage(ctx, WithSort("Title", false), WithLimit(3))
	if err != nil {
		t.Fatalf("ScanPage page1: %v", err)
	}

	if len(page1) != 3 {
		t.Fatalf("expected 3 items on page1, got %d", len(page1))
	}

	if cursor1 == nil {
		t.Fatal("expected non-nil cursor after page1")
	}

	// Encode the cursor for HTTP transport.
	encoded, err := cursor1.Encode()
	if err != nil {
		t.Fatalf("cursor.Encode: %v", err)
	}

	if encoded == "" {
		t.Fatal("expected non-empty encoded cursor")
	}

	// Verify it's base64 (HTTP-safe, no special chars).
	for _, c := range encoded {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' &&
			c != '_' {
			t.Fatalf("encoded cursor contains non-base64 char: %q in %q", c, encoded)
		}
	}

	// Page 2 via WithCursorString (the HTTP-safe path).
	page2a, cursor2a, err := reader.ScanPage(
		ctx,
		WithSort("Title", false),
		WithLimit(3),
		WithCursorString(encoded),
	)
	if err != nil {
		t.Fatalf("ScanPage page2a (WithCursorString): %v", err)
	}

	if len(page2a) != 3 {
		t.Fatalf("expected 3 items on page2a, got %d", len(page2a))
	}

	// Page 2 via WithCursor(cursor1.Value) (the raw value path).
	page2b, cursor2b, err := reader.ScanPage(
		ctx,
		WithSort("Title", false),
		WithLimit(3),
		WithCursor(cursor1.Value),
	)
	if err != nil {
		t.Fatalf("ScanPage page2b (WithCursor): %v", err)
	}

	if len(page2b) != 3 {
		t.Fatalf("expected 3 items on page2b, got %d", len(page2b))
	}

	// Both paths should return the same items (same cursor, same data).
	for i := range page2a {
		if page2a[i].ID != page2b[i].ID {
			t.Errorf("page2 mismatch at %d: WithCursorString=%s, WithCursor=%s",
				i, page2a[i].ID, page2b[i].ID)
		}
	}

	// Both should produce the same next cursor.
	if cursor2a == nil || cursor2b == nil {
		t.Fatal("expected non-nil cursors after page2")
	}

	if cursor2a.Value != cursor2b.Value {
		t.Errorf("cursor mismatch: WithCursorString=%v, WithCursor=%v",
			cursor2a.Value, cursor2b.Value)
	}

	// Verify no overlap between pages.
	seen := make(map[testTaskID]bool)
	for _, item := range page1 {
		seen[item.ID] = true
	}

	for _, item := range page2a {
		if seen[item.ID] {
			t.Errorf("item %s appeared on both pages", item.ID)
		}
	}
}

func TestPrefetchCache_WithCursorString_CacheHit(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	for i := range 8 {
		_ = store.Apply(ctx, "task_created", testTask{
			ID:    testTaskID(fmt.Sprintf("ch%d", i)),
			Title: fmt.Sprintf("Cache-%02d", i),
		})
	}

	// Page 1: triggers prefetch population.
	_, cursor1, err := reader.ScanPage(ctx, WithSort("Title", false), WithLimit(3))
	if err != nil {
		t.Fatalf("ScanPage page1: %v", err)
	}

	encoded, err := cursor1.Encode()
	if err != nil {
		t.Fatalf("cursor.Encode: %v", err)
	}

	// Page 2 via WithCursorString — should be a cache hit.
	page2, _, err := reader.ScanPage(
		ctx,
		WithSort("Title", false),
		WithLimit(3),
		WithCursorString(encoded),
	)
	if err != nil {
		t.Fatalf("ScanPage page2: %v", err)
	}

	if len(page2) != 3 {
		t.Fatalf("expected 3 items on page2, got %d", len(page2))
	}

	// Verify cache has the entry (not just that the scan worked).
	// The key should be the encoded cursor.
	expectedKey := prefetchCursorKey("tasks", cursor1)
	if cache.Get(expectedKey) == nil {
		t.Error("expected cache hit for encoded cursor key")
	}

	// Also verify the WithCursor path hits the same cache entry.
	page2raw, _, err := reader.ScanPage(
		ctx,
		WithSort("Title", false),
		WithLimit(3),
		WithCursor(cursor1.Value),
	)
	if err != nil {
		t.Fatalf("ScanPage page2raw: %v", err)
	}

	if len(page2raw) != 3 {
		t.Fatalf("expected 3 items on page2raw, got %d", len(page2raw))
	}

	// Both paths should return the same items.
	for i := range page2 {
		if page2[i].ID != page2raw[i].ID {
			t.Errorf("item mismatch at %d: WithCursorString=%s, WithCursor=%s",
				i, page2[i].ID, page2raw[i].ID)
		}
	}
}

func TestWithCursorString_EmptyString(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "e1", Title: "Empty"})

	// Empty string → nil cursor → should not crash, should return all.
	page, _, err := reader.ScanPage(ctx, WithSort("Title", false), WithLimit(10),
		WithCursorString(""))
	if err != nil {
		t.Fatalf("ScanPage with empty cursor string: %v", err)
	}

	if len(page) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page))
	}
}

func TestWithCursorString_InvalidString(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	cache := NewPrefetchCache()
	reader := NewReader[testTask](store, "tasks").WithPrefetch(cache)

	ctx := context.Background()
	_ = store.Apply(ctx, "task_created", testTask{ID: "i1", Title: "Invalid"})

	// Invalid base64 string → ParseCursor fails → cursor stays nil → no crash.
	page, _, err := reader.ScanPage(ctx, WithSort("Title", false), WithLimit(10),
		WithCursorString("!!!invalid!!!"))
	if err != nil {
		t.Fatalf("ScanPage with invalid cursor string: %v", err)
	}

	if len(page) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page))
	}
}

// --- PrefetchCache concurrent access test ---

func TestPrefetchCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache := NewPrefetchCache()

	var wg sync.WaitGroup

	// Writers: hammer Put from multiple goroutines.
	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range 500 {
				cache.Put(fmt.Sprintf("key-%d", i%20), []any{fmt.Sprintf("val-%d", i)})
			}
		}()
	}

	// Readers: hammer Get concurrently.
	for range 4 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range 1000 {
				_ = cache.Get(fmt.Sprintf("key-%d", i%20))
			}
		}()
	}

	// Clearer: periodically wipes the cache.
	wg.Add(1)

	go func() {
		defer wg.Done()

		for range 100 {
			cache.Clear()
		}
	}()

	wg.Wait()

	// Final state should be usable.
	cache.Put("final", []any{"done"})

	if v := cache.Get("final"); v == nil {
		t.Error("expected cache hit after concurrent writes")
	}
}

// --- SQLite engine SSE replay test ---

func TestSSE_LastEventID_Reconnect_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatal(err)
	}

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher[testTask](store, "tasks")
	watcher.WithReplay(100)
	defer watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(w, r, watcher, WithSSETimeout(5*time.Second))
	})

	srv := &http.Server{Handler: mux}
	defer srv.Close()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv.Addr = ln.Addr().String()
	go srv.Serve(ln)

	time.Sleep(100 * time.Millisecond)

	// Phase 1: Connect and receive events.
	conn1, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn1.Close()

	_, _ = conn1.Write([]byte("GET /events HTTP/1.0\r\nHost: localhost\r\n\r\n"))
	time.Sleep(200 * time.Millisecond)

	_ = store.Apply(ctx, "task_created", testTask{ID: "sc1", Title: "SQLite Reconnect 1"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "sc2", Title: "SQLite Reconnect 2"})

	var data1 string

	deadline := time.Now().Add(5 * time.Second)
	rbuf := make([]byte, 8192)

	for time.Now().Before(deadline) {
		conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn1.Read(rbuf)
		if err != nil {
			break
		}

		data1 += string(rbuf[:n])

		if strings.Contains(data1, "SQLite Reconnect 2") {
			break
		}
	}

	if !strings.Contains(data1, "id: 1") {
		t.Errorf("expected 'id: 1' in SSE output, got: %s", data1)
	}

	if !strings.Contains(data1, "id: 2") {
		t.Errorf("expected 'id: 2' in SSE output, got: %s", data1)
	}

	// Phase 2: Reconnect with Last-Event-ID: 2.
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	_ = store.Apply(ctx, "task_created", testTask{ID: "sc3", Title: "SQLite Reconnect 3"})
	_ = store.Apply(ctx, "task_created", testTask{ID: "sc4", Title: "SQLite Reconnect 4"})

	time.Sleep(100 * time.Millisecond)

	conn2, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	defer conn2.Close()

	_, _ = conn2.Write(
		[]byte("GET /events HTTP/1.0\r\nHost: localhost\r\nLast-Event-ID: 2\r\n\r\n"),
	)

	var data2 string

	deadline2 := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline2) {
		conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn2.Read(rbuf)
		if err != nil {
			break
		}

		data2 += string(rbuf[:n])

		if strings.Contains(data2, "SQLite Reconnect 3") &&
			strings.Contains(data2, "SQLite Reconnect 4") {
			break
		}
	}

	if !strings.Contains(data2, "id: 3") {
		t.Errorf("expected 'id: 3' in replayed output, got: %s", data2)
	}

	if !strings.Contains(data2, "id: 4") {
		t.Errorf("expected 'id: 4' in replayed output, got: %s", data2)
	}

	if strings.Contains(data2, "SQLite Reconnect 1") {
		t.Errorf("replay should not include event 1 (before Last-Event-ID)")
	}
}
