package metaengine_test

// A15 conformance: layout-plan stamping is ON by default — column-add and
// column-rename each trip the drift WARN on the next boot over the same
// engine — and WithLayoutStampsDisabled removes both the warning and the
// stamp writes. Scenario: two Plan() boots over ONE SQLite engine (the
// stamp persists in the engine's map storage across boots, exactly like a
// process restart over a real database).

import (
	"bytes"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type stampQueryInput struct{}

type stampResult struct {
	ID       string
	Status   string
	Priority int
}

type stampEvent struct {
	ID       string
	Status   string
	Priority int
}

// stampQueryFields declares the layout surface: filter on "status", plus
// the given extra declarative fields (the planned-table columns come from
// the declared filter/sort fields — adding or changing one IS the column
// drift the stamp must catch).
type stampQueryFields struct {
	name string
	args []any
}

func stampQuery(fields stampQueryFields) metaengine.QueryDecl[stampQueryInput, stampResult] {
	base := []any{
		metaengine.OnRecord(
			stampEvent{},
			func(_ record.Record, e stampEvent) (string, stampResult) {
				return e.ID, stampResult{ID: e.ID, Status: e.Status, Priority: e.Priority}
			},
		),
		metaengine.FilterOnField[stampResult]("status", metaengine.FilterEq),
	}

	return metaengine.Query[stampQueryInput, stampResult](fields.name, append(base, fields.args...)...)
}

// The three declarations share the query name (hence the collection): one
// filter column; a second sort COLUMN ADDED; the filter COLUMN CHANGED.
var stampFieldsBase = stampQueryFields{name: "stamp_tasks"}

var stampFieldsColumnAdded = stampQueryFields{
	name: "stamp_tasks",
	args: []any{metaengine.SortOnField[stampResult]("priority", true)},
}

var stampFieldsColumnChanged = stampQueryFields{
	name: "stamp_tasks",
	args: []any{metaengine.SortOnField[stampResult]("id", true)},
}

// openStampEngine opens a shared in-memory SQLite engine (one connection so
// the :memory: database survives across Store boots). The database name is
// run-unique: cache=shared namespaces :memory: databases by name, so a
// fixed name would let PARALLEL tests' boots drift-warn against each other.
func openStampEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	dsn := fmt.Sprintf("file:stamp_boot_%d?mode=memory&cache=shared", time.Now().UnixNano())

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// captureWarns swaps the default slog logger for one writing into buf and
// restores it on cleanup.
func captureWarns(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return buf
}

// mustBoot runs one Plan() boot over the engine. Store.Close would close the
// ENGINE too (stores own their engines), so the store is closed at test
// cleanup instead — after both boots used the shared engine.
func mustBoot(t *testing.T, eng metaengine.Engine, args ...any) {
	t.Helper()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, args...)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })
}

// TestLayoutStamp_ColumnAddTripsDriftWarn pins the default-ON contract: the
// next boot over the same engine with an ADDED result column warns (stored
// rows predate the wider schema).
func TestLayoutStamp_ColumnAddTripsDriftWarn(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, stampQuery(stampFieldsBase))

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("initial boot must not warn: %s", buf.String())
	}

	// A sort column ADDED to the declaration = a new planned-table column.
	mustBoot(t, eng, stampQuery(stampFieldsColumnAdded))

	if !strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("column-add boot must warn: %s", buf.String())
	}
}

// TestLayoutStamp_ColumnRenameTripsDriftWarn pins the rename case: a renamed
// result column is a fingerprint change (remove + add), so the next boot
// warns too.
func TestLayoutStamp_ColumnRenameTripsDriftWarn(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, stampQuery(stampFieldsBase))
	buf.Reset()

	// A DIFFERENT sort column = the planned table's column set changed
	// (remove + add — the rename shape).
	mustBoot(t, eng, stampQuery(stampFieldsColumnChanged))

	if !strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("column-change boot must warn: %s", buf.String())
	}
}

// TestLayoutStamp_SamePlanDoesNotWarn pins the quiet path: rebooting with the
// IDENTICAL declaration never warns (every process start re-registers the
// same queries).
func TestLayoutStamp_SamePlanDoesNotWarn(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, stampQuery(stampFieldsBase))
	buf.Reset()
	mustBoot(t, eng, stampQuery(stampFieldsBase))

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("identical reboot must not warn: %s", buf.String())
	}
}

// TestLayoutStamp_DisabledOptsOut pins the config inversion: with
// WithLayoutStampsDisabled, drift stays silent.
func TestLayoutStamp_DisabledOptsOut(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, metaengine.WithLayoutStampsDisabled(), stampQuery(stampFieldsBase))
	mustBoot(t, eng, metaengine.WithLayoutStampsDisabled(), stampQuery(stampFieldsColumnAdded))

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("disabled stamping must stay silent: %s", buf.String())
	}
}
