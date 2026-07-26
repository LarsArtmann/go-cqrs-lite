package relational

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// messagesRow is the minimal `messages` row used by every UpsertCols/UpsertExpr
// scenario. Each test case overrides only the fields that matter for its
// assertion (id + content + maybe author/created_at) and reuses the rest.
func messagesRow(id, content string) Row {
	return Row{
		"id":         id,
		"channel_id": "c1",
		"guild_id":   "g1",
		"author_id":  "u1",
		"content":    content,
		"created_at": "2026-01-01T00:00:00Z",
	}
}

// upsertScenario captures the two-phase upsert flow shared by every test in
// this file: an initial row is inserted (action=initial), then a follow-up
// operation (action=op) runs against it. The verify callback queries the
// resulting row and asserts the expected state.
type upsertScenario struct {
	name     string
	projName string
	id       string
	initial  Row
	insertOp upsertOp // runs FIRST against the empty table
	followOp upsertOp // runs SECOND against the row inserted by insertOp
	verify   func(t *testing.T, db *sql.DB, ctx context.Context)
}

// upsertOp discriminates Upsert / UpsertCols / UpsertExpr — the three sink
// operations exercised by this file. Each variant carries exactly the
// arguments that operation requires.
type upsertOp struct {
	kind  string // "upsert", "cols", "expr"
	row   Row
	cols  []string  // UpsertCols
	exprs []SetExpr // UpsertExpr
}

func (o upsertOp) apply(ctx context.Context, sink ProjectionSink) error {
	switch o.kind {
	case "upsert":
		return sink.Upsert(ctx, "messages", o.row)
	case "cols":
		return sink.UpsertCols(ctx, "messages", o.row, o.cols)
	case "expr":
		return sink.UpsertExpr(ctx, "messages", o.row, o.exprs)
	}
	panic("upsertTest: unknown op kind " + o.kind)
}

// runUpsertScenario executes the full two-phase flow for one entry: builds the
// projection, runs the initial op, runs the follow-up op, then queries the
// row and asserts the expected state. Each scenario runs in isolation because
// every entry uses a unique id.
func runUpsertScenario(t *testing.T, sc upsertScenario) {
	t.Helper()

	schema := discordSchema()

	db, ctx := openRelationalCtx(t)

	buildProj := func(name string, op upsertOp) *RelationalProjection {
		t.Helper()

		proj, err := NewRelationalProjection(name, schema, db, sqlpkg.SQLiteDialect{},
			func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
				return op.apply(ctx, sink)
			}, nil)
		if err != nil {
			t.Fatalf("new projection %s: %v", name, err)
		}

		return proj
	}

	initial := buildProj(sc.projName+"-init", sc.insertOp)
	if err := initial.Handle(ctx, newEvent(t, "Init", nil)); err != nil {
		t.Fatalf("[%s] handle init: %v", sc.name, err)
	}

	if sc.followOp.kind == "" {
		// Fresh-insert scenarios only run the initial op.
		sc.verify(t, db, ctx)

		return
	}

	follow := buildProj(sc.projName+"-op", sc.followOp)
	if err := follow.Handle(ctx, newEvent(t, "Op", nil)); err != nil {
		t.Fatalf("[%s] handle op: %v", sc.name, err)
	}

	sc.verify(t, db, ctx)
}

// queryMessageCol reads a single string column from the messages table.
func queryMessageCol(t *testing.T, db *sql.DB, ctx context.Context, id, col string) string {
	t.Helper()

	var val string

	if err := db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM messages WHERE id = ?", col), id,
	).Scan(&val); err != nil {
		t.Fatalf("query %s for %s: %v", col, id, err)
	}

	return val
}

func TestSinkUpsert(t *testing.T) {
	t.Parallel()

	// COALESCE expr: empty new content should preserve original.
	emptyContentExpr := []SetExpr{{
		Column: "content",
		Expr:   "COALESCE(NULLIF(excluded.content, ''), messages.content)",
	}}

	scenarios := []upsertScenario{
		{
			name:     "UpsertCols_PartialUpdateOnly",
			projName: "upsert-cols",
			id:       "m1",
			initial:  messagesRow("m1", "original"),
			insertOp: upsertOp{kind: "upsert", row: messagesRow("m1", "original")},
			followOp: upsertOp{
				kind: "cols",
				row:  messagesRow("m1", "edited"),
				cols: []string{"content"},
			},
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m1", "content"); got != "edited" {
					t.Fatalf("content: got %q, want %q", got, "edited")
				}
				if got := queryMessageCol(t, db, ctx, "m1", "author_id"); got != "u1" {
					t.Fatalf("author_id: got %q, want %q", got, "u1")
				}
				if got := queryMessageCol(t, db, ctx, "m1", "created_at"); got != "2026-01-01T00:00:00Z" {
					t.Fatalf("created_at: got %q, want %q", got, "2026-01-01T00:00:00Z")
				}
			},
		},
		{
			name:     "UpsertCols_EmptyUpdateColsDoesNothing",
			projName: "upsert-cols-empty",
			id:       "m2",
			initial:  messagesRow("m2", "original"),
			insertOp: upsertOp{kind: "upsert", row: messagesRow("m2", "original")},
			followOp: upsertOp{
				kind: "cols",
				row: Row{
					"id":         "m2",
					"channel_id": "c1",
					"guild_id":   "g1",
					"author_id":  "DIFFERENT",
					"content":    "should-not-apply",
					"created_at": "2099-01-01T00:00:00Z",
				},
			},
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m2", "content"); got != "original" {
					t.Fatalf("content: got %q, want %q", got, "original")
				}
			},
		},
		{
			name:     "UpsertExpr_COALESCE_PreservesEmpty",
			projName: "upsert-expr",
			id:       "m3",
			initial:  messagesRow("m3", "original content"),
			insertOp: upsertOp{kind: "upsert", row: messagesRow("m3", "original content")},
			followOp: upsertOp{
				kind:  "expr",
				row:   messagesRow("m3", ""),
				exprs: emptyContentExpr,
			},
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m3", "content"); got != "original content" {
					t.Fatalf("content: got %q, want %q", got, "original content")
				}
			},
		},
		{
			name:     "UpsertExpr_COALESCE_UpdatesNonEmpty",
			projName: "upsert-expr3",
			id:       "m4",
			initial:  messagesRow("m4", "old content"),
			insertOp: upsertOp{kind: "upsert", row: messagesRow("m4", "old content")},
			followOp: upsertOp{
				kind:  "expr",
				row:   messagesRow("m4", "new content"),
				exprs: emptyContentExpr,
			},
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m4", "content"); got != "new content" {
					t.Fatalf("content: got %q, want %q", got, "new content")
				}
			},
		},
		{
			name:     "UpsertExpr_EmptyExprsDoesNothing",
			projName: "upsert-expr5",
			id:       "m5",
			initial:  messagesRow("m5", "untouched"),
			insertOp: upsertOp{kind: "upsert", row: messagesRow("m5", "untouched")},
			followOp: upsertOp{
				kind: "expr",
				row: Row{
					"id":         "m5",
					"channel_id": "c1",
					"guild_id":   "g1",
					"author_id":  "DIFFERENT",
					"content":    "should-not-apply",
					"created_at": "2099-01-01T00:00:00Z",
				},
			},
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m5", "content"); got != "untouched" {
					t.Fatalf("content: got %q, want %q", got, "untouched")
				}
			},
		},
		{
			name:     "UpsertExpr_BoundArgsAppends",
			projName: "upsert-expr-args",
			id:       "m6",
			initial:  messagesRow("m6", "hello"),
			insertOp: upsertOp{kind: "upsert", row: messagesRow("m6", "hello")},
			followOp: upsertOp{
				kind: "expr",
				row:  messagesRow("m6", "ignored-on-conflict"),
				exprs: []SetExpr{{
					Column: "content",
					Expr:   "messages.content || ?",
					Args:   []any{" world"},
				}},
			},
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m6", "content"); got != "hello world" {
					t.Fatalf("content: got %q, want %q", got, "hello world")
				}
			},
		},
		{
			// UpsertCols on a row that does not exist yet: the INSERT path
			// runs and writes ALL provided columns, regardless of updateCols
			// (which only governs the ON CONFLICT DO UPDATE branch).
			name:     "UpsertCols_FreshInsert",
			projName: "upsert-cols-fresh",
			id:       "m7",
			insertOp: upsertOp{
				kind: "cols",
				row:  messagesRow("m7", "fresh-insert"),
				cols: []string{"content"},
			},
			followOp: upsertOp{}, // no follow-up needed for fresh insert
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m7", "content"); got != "fresh-insert" {
					t.Fatalf("content: got %q, want %q", got, "fresh-insert")
				}
				if got := queryMessageCol(t, db, ctx, "m7", "author_id"); got != "u1" {
					t.Fatalf("author_id: got %q, want %q", got, "u1")
				}
				if got := queryMessageCol(t, db, ctx, "m7", "channel_id"); got != "c1" {
					t.Fatalf("channel_id: got %q, want %q", got, "c1")
				}
			},
		},
		{
			// UpsertExpr on a row that does not exist yet: the INSERT path
			// runs and writes ALL provided columns; SetExprs are only
			// evaluated on conflict.
			name:     "UpsertExpr_FreshInsert",
			projName: "upsert-expr-fresh",
			id:       "m8",
			insertOp: upsertOp{
				kind: "expr",
				row:  messagesRow("m8", "fresh-expr-insert"),
				exprs: []SetExpr{{
					Column: "content",
					Expr:   "messages.content || ?",
					Args:   []any{" should-not-apply"},
				}},
			},
			followOp: upsertOp{}, // no follow-up needed for fresh insert
			verify: func(t *testing.T, db *sql.DB, ctx context.Context) {
				t.Helper()
				if got := queryMessageCol(t, db, ctx, "m8", "content"); got != "fresh-expr-insert" {
					t.Fatalf("content: got %q, want %q", got, "fresh-expr-insert")
				}
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			runUpsertScenario(t, sc)
		})
	}
}
