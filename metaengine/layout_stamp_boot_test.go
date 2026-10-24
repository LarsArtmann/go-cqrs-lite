package metaengine_test

// A15 conformance: layout-plan stamping is ON by default — column-add and
// column-rename each trip the drift WARN on re-apply — and
// WithLayoutStampsDisabled removes both the warning and the stamp writes.

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type stampTaskV1 struct {
	ID       string
	Priority int
}

type stampTaskColumnAdded struct {
	ID       string
	Priority int
	Note     string
}

type stampTaskColumnRenamed struct {
	ID       string
	Priority int
	Memo     string
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

func stampPlanFor(t *testing.T, collection string, resultType any) metaengine.LayoutPlan {
	t.Helper()

	plan := metaengine.BuildLayoutPlanForType(collection, nil, nil, typeOf(resultType))
	if len(plan.Columns) == 0 {
		t.Fatalf("no columns derived for %T", resultType)
	}

	return plan
}

// TestLayoutStamp_ColumnAddTripsDriftWarn pins the default-ON contract: the
// same collection re-applied with an ADDED column warns (the stored rows
// predate the wider schema) and re-records the new fingerprint.
func TestLayoutStamp_ColumnAddTripsDriftWarn(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)

	engine := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{engine})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	first := stampPlanFor(t, "stamp_tasks", stampTaskV1{})
	store.StampLayoutPlanForTest(t, first)

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("initial apply must not warn: %s", buf.String())
	}

	second := stampPlanFor(t, "stamp_tasks", stampTaskColumnAdded{})
	store.StampLayoutPlanForTest(t, second)

	if !strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("column-add re-apply must warn: %s", buf.String())
	}
}

// TestLayoutStamp_ColumnRenameTripsDriftWarn pins the rename case: a renamed
// column is a fingerprint change (remove + add), so the re-apply warns too.
func TestLayoutStamp_ColumnRenameTripsDriftWarn(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)

	engine := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{engine})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	store.StampLayoutPlanForTest(t, stampPlanFor(t, "stamp_tasks", stampTaskColumnAdded{}))
	buf.Reset()

	store.StampLayoutPlanForTest(t, stampPlanFor(t, "stamp_tasks", stampTaskColumnRenamed{}))

	if !strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("column-rename re-apply must warn: %s", buf.String())
	}
}

// TestLayoutStamp_DisabledOptsOut pins the config inversion: with
// WithLayoutStampsDisabled, drift stays silent and nothing is recorded.
func TestLayoutStamp_DisabledOptsOut(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)

	engine := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan(
		[]metaengine.Engine{engine},
		metaengine.WithLayoutStampsDisabled(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	store.StampLayoutPlanForTest(t, stampPlanFor(t, "stamp_tasks", stampTaskV1{}))
	store.StampLayoutPlanForTest(t, stampPlanFor(t, "stamp_tasks", stampTaskColumnAdded{}))

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("disabled stamping must stay silent: %s", buf.String())
	}
}

// TestLayoutStamp_SamePlanDoesNotWarn pins the quiet path: re-applying the
// IDENTICAL plan never warns (boot idempotence — every process start
// re-registers the same queries).
func TestLayoutStamp_SamePlanDoesNotWarn(t *testing.T) {
	t.Parallel()

	buf := captureWarns(t)

	engine := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{engine})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	store.StampLayoutPlanForTest(t, stampPlanFor(t, "stamp_tasks", stampTaskV1{}))
	buf.Reset()
	store.StampLayoutPlanForTest(t, stampPlanFor(t, "stamp_tasks", stampTaskV1{}))

	if strings.Contains(buf.String(), "layout stamp drift") {
		t.Fatalf("identical re-apply must not warn: %s", buf.String())
	}
}

var _ = context.Background
