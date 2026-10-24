package sqliteengine_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// openEncodingEngine returns an engine on its own private in-memory database so
// these tests can run in parallel without sharing rows.
func openEncodingEngine(
	t *testing.T,
	opts ...sqliteengine.EngineOption,
) metaengine.Engine {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())

	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db, opts...)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// assertRejection also pins the error CODE, so a later refactor cannot move
// the failure to a different stage while the family stays Rejection.
func assertRejection(t *testing.T, err error, wantCode string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("error %v classified as %v, want Rejection", err, family)
	}

	if code := errorfamily.Code(err); code != wantCode {
		t.Fatalf("error code = %q, want %q (%v)", code, wantCode, err)
	}
}

const (
	mapEncodeCode    = "sqliteengine.encode_value"
	streamEncodeCode = "metaengine.encode_stream_value"
)

// TestSQLite_MapSet_RejectsUnrepresentableValue pins the removal of the
// fmt.Sprintf("%v") fallback: an unencodable value must fail loudly and must
// not leave a row behind.
func TestSQLite_MapSet_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	assertRejection(t, mb.MapSet(ctx, "tasks", "task-1", make(chan int)), mapEncodeCode)

	_, exists, err := mb.MapGet(ctx, "tasks", "task-1")
	if err != nil {
		t.Fatalf("MapGet after rejected write: %v", err)
	}

	if exists {
		t.Fatal("MapSet stored a row despite the encode failure")
	}
}

func TestSQLite_MapSet_PlannedTable_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)

	plan := metaengine.BuildLayoutPlan("planned_items", []string{"status"}, nil)
	if err := eng.(metaengine.LayoutPlanApplier).ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	assertRejection(t, mb.MapSet(ctx, "planned_items", "k-1", func() {}), mapEncodeCode)

	_, exists, err := mb.MapGet(ctx, "planned_items", "k-1")
	if err != nil {
		t.Fatalf("MapGet after rejected write: %v", err)
	}

	if exists {
		t.Fatal("planned MapSet stored a row despite the encode failure")
	}
}

func TestSQLite_MapUpdate_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	if err := mb.MapSet(ctx, "tasks", "task-1", map[string]any{"n": 1}); err != nil {
		t.Fatalf("seed MapSet: %v", err)
	}

	updater := eng.(metaengine.MapUpdater)

	err := updater.MapUpdate(ctx, "tasks", "task-1", func(any) any {
		return make(chan int)
	})
	assertRejection(t, err, mapEncodeCode)

	got, exists, err := mb.MapGet(ctx, "tasks", "task-1")
	if err != nil {
		t.Fatalf("MapGet after rejected update: %v", err)
	}

	if !exists {
		t.Fatal("MapUpdate deleted the previously stored row")
	}

	row, ok := got.(map[string]any)
	if !ok || row["n"] != float64(1) {
		t.Fatalf("stored value = %#v, want the untouched seed row", got)
	}
}

func TestSQLite_MapSetAt_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t, sqliteengine.WithCellVersioning())
	ctx := context.Background()
	versioned := eng.(metaengine.VersionedWriter)

	assertRejection(t, versioned.MapSetAt(ctx, "tasks", "task-1", make(chan int), time.Now()), mapEncodeCode)

	mb := eng.(metaengine.MapBackend)

	_, exists, err := mb.MapGet(ctx, "tasks", "task-1")
	if err != nil {
		t.Fatalf("MapGet after rejected write: %v", err)
	}

	if exists {
		t.Fatal("MapSetAt stored a latest row despite the encode failure")
	}
}

// TestSQLite_StreamAppend_RejectsBatchAtomically pins the all-or-nothing
// contract: the whole batch is encoded before any row is appended.
func TestSQLite_StreamAppend_RejectsBatchAtomically(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	sl := eng.(metaengine.StreamLogBackend)

	err := sl.StreamAppend(ctx, "streams", "s-1", []any{"ok", make(chan int)})
	assertRejection(t, err, streamEncodeCode)

	version, err := sl.StreamVersion(ctx, "streams", "s-1")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	if version != 0 {
		t.Fatalf("stream version = %d after a rejected batch, want 0", version)
	}
}

func TestSQLite_StreamAppendExpected_RejectsBatchAtomically(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	ap := eng.(metaengine.AtomicAppender)

	err := ap.StreamAppendExpected(ctx, "streams", "s-1", 0, []any{"ok", make(chan int)})
	assertRejection(t, err, streamEncodeCode)

	version, err := eng.(metaengine.StreamLogBackend).StreamVersion(ctx, "streams", "s-1")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	if version != 0 {
		t.Fatalf("stream version = %d after a rejected batch, want 0", version)
	}
}

func TestSQLite_LogAppend_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	log := eng.(metaengine.LogBackend)

	assertRejection(t, log.LogAppend(ctx, "events", make(chan int)), streamEncodeCode)

	tail, err := log.LogTail(ctx, "events", 10)
	if err != nil {
		t.Fatalf("LogTail: %v", err)
	}

	if len(tail) != 0 {
		t.Fatalf("log holds %d entries after a rejected append, want 0", len(tail))
	}
}

func TestSQLite_MultiAdd_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	mm := eng.(metaengine.MultimapBackend)

	assertRejection(t, mm.MultiAdd(ctx, "tags", "k-1", make(chan int)), streamEncodeCode)

	values, err := mm.MultiGet(ctx, "tags", "k-1")
	if err != nil {
		t.Fatalf("MultiGet: %v", err)
	}

	if len(values) != 0 {
		t.Fatalf("multimap holds %d values after a rejected add, want 0", len(values))
	}
}

// TestSQLite_RepresentableValuesStillStore guards against over-tightening:
// the rejection must only fire for values JSON cannot carry.
func TestSQLite_RepresentableValuesStillStore(t *testing.T) {
	t.Parallel()

	eng := openEncodingEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	if err := mb.MapSet(ctx, "tasks", "task-1", map[string]any{"ok": true}); err != nil {
		t.Fatalf("representable value rejected: %v", err)
	}

	got, exists, err := mb.MapGet(ctx, "tasks", "task-1")
	if err != nil {
		t.Fatalf("MapGet: %v", err)
	}

	if !exists {
		t.Fatal("representable value was not stored")
	}

	row, ok := got.(map[string]any)
	if !ok || row["ok"] != true {
		t.Fatalf("stored value = %#v, want map[ok:true]", got)
	}
}
