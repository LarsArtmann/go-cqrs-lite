package metaengine_test

// A15 conformance: layout-plan stamping is ON by default — a SILENT plan
// drift (same column names, changed column TYPES: the name-typing-kill
// corruption class, where "priority" was TEXT and the declaration now says
// BIGINT) WARNs on the next boot over the same engine — and
// WithLayoutStampsDisabled removes both the warning and the stamp writes.
//
// Boundary, pinned by the engine itself: a column-SET conflict (added or
// renamed columns) never reaches the stamp — the engine's ApplyLayoutPlan
// hard-errors on the conflicting plan first, which is the stronger guarantee
// for that class. The stamp covers exactly what the engine lets through
// silently.

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

// stampResultStringPriority / stampResultIntPriority share field NAMES; the
// Priority TYPE differs — the planned column set stays [priority status]
// while the column TYPES drift (TEXT vs BIGINT after BuildLayoutPlanForType).
type stampResultStringPriority struct {
	ID       string
	Status   string
	Priority string
}

type stampResultIntPriority struct {
	ID       string
	Status   string
	Priority int
}

type stampEvent struct {
	ID       string
	Status   string
	Priority int
}

func stampQueryStringPriority() metaengine.QueryDecl[stampQueryInput, stampResultStringPriority] {
	return metaengine.Query[stampQueryInput, stampResultStringPriority](
		"stamp_tasks",
		metaengine.OnRecord(
			stampEvent{},
			func(_ record.Record, e stampEvent) (string, stampResultStringPriority) {
				return e.ID, stampResultStringPriority{ID: e.ID, Status: e.Status}
			},
		),
		metaengine.FilterOnField[stampResultStringPriority]("status", metaengine.FilterEq),
		metaengine.SortOnField[stampResultStringPriority]("priority", true),
	)
}

func stampQueryIntPriority() metaengine.QueryDecl[stampQueryInput, stampResultIntPriority] {
	return metaengine.Query[stampQueryInput, stampResultIntPriority](
		"stamp_tasks",
		metaengine.OnRecord(
			stampEvent{},
			func(_ record.Record, e stampEvent) (string, stampResultIntPriority) {
				return e.ID, stampResultIntPriority(e)
			},
		),
		metaengine.FilterOnField[stampResultIntPriority]("status", metaengine.FilterEq),
		metaengine.SortOnField[stampResultIntPriority]("priority", true),
	)
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

// TestLayoutStamp_ColumnTypeDriftTripsWarn pins the default-ON contract: the
// engine applies a same-names/different-types plan SILENTLY, and the stamp
// is what surfaces it — the next boot warns that stored rows predate the
// current schema. These tests are deliberately NOT t.Parallel: they swap
// the process-global slog default to capture warnings, and parallel
// siblings swapping it back would race the capture.
func TestLayoutStamp_ColumnTypeDriftTripsWarn(t *testing.T) {
	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, stampQueryStringPriority())

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("initial boot must not warn: %s", buf.String())
	}

	mustBoot(t, eng, stampQueryIntPriority())

	if !strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("type-drift boot must warn: %s", buf.String())
	}
}

// TestLayoutStamp_SamePlanDoesNotWarn pins the quiet path: rebooting with the
// IDENTICAL declaration never warns (every process start re-registers the
// same queries). Sequential for the same logger-capture reason.
func TestLayoutStamp_SamePlanDoesNotWarn(t *testing.T) {
	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, stampQueryStringPriority())
	buf.Reset()
	mustBoot(t, eng, stampQueryStringPriority())

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("identical reboot must not warn: %s", buf.String())
	}
}

// TestLayoutStamp_DisabledOptsOut pins the config inversion: with
// WithLayoutStampsDisabled, drift stays silent. Sequential for the same
// logger-capture reason.
func TestLayoutStamp_DisabledOptsOut(t *testing.T) {
	buf := captureWarns(t)
	eng := openStampEngine(t)

	mustBoot(t, eng, metaengine.WithLayoutStampsDisabled(), stampQueryStringPriority())
	mustBoot(t, eng, metaengine.WithLayoutStampsDisabled(), stampQueryIntPriority())

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("disabled stamping must stay silent: %s", buf.String())
	}
}
