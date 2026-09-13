package sqliteengine_test

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type exportTask struct {
	ID    string
	Title string
}

type exportTaskInput struct {
	ID string
}

func exportTaskQuery() metaengine.QueryDecl[exportTaskInput, exportTask] {
	return metaengine.Query[exportTaskInput, exportTask](
		"export_stream_tasks",
		metaengine.OnRecordTyped(
			"export_stream_task_created",
			exportTask{},
			func(_ record.Record, e exportTask) (string, exportTask) {
				return e.ID, e
			},
		),
	)
}

// TestExport_OverSQLiteEngine_ContainsAllRows exercises the real streaming
// path: the sqlite engine implements StreamingScan, and Store.Export routes
// through Store.StreamCollection, so this test covers the production-shaped
// export instead of the synthetic wrapper used in the metaengine unit test.
func TestExport_OverSQLiteEngine_ContainsAllRows(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, exportTaskQuery())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	for _, task := range []exportTask{
		{ID: "t1", Title: "First"},
		{ID: "t2", Title: "Second"},
		{ID: "t3", Title: "Third"},
	} {
		if err := store.Apply(ctx, "export_stream_task_created", task); err != nil {
			t.Fatalf("apply %s: %v", task.ID, err)
		}
	}

	var buf bytes.Buffer

	if err := store.Export(ctx, &buf); err != nil {
		t.Fatalf("export: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, `"export_stream_tasks":[`) {
		t.Fatalf("export missing collection array: %s", out)
	}

	for _, title := range []string{"First", "Second", "Third"} {
		if !strings.Contains(out, title) {
			t.Fatalf("export missing row %q: %s", title, out)
		}
	}
}
