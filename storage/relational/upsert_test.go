package relational

import (
	"context"
	"database/sql"
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

				var content, author, created string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content, author_id, created_at FROM messages WHERE id = 'm1'",
				).Scan(&content, &author, &created); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "edited" {
					t.Fatalf("content should be updated to 'edited', got %q", content)
				}
				if author != "u1" {
					t.Fatalf("author_id should be preserved as 'u1', got %q", author)
				}
				if created != "2026-01-01T00:00:00Z" {
					t.Fatalf("created_at should be preserved, got %q", created)
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

				var content string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content FROM messages WHERE id = 'm2'",
				).Scan(&content); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "original" {
					t.Fatalf(
						"empty updateCols should do nothing, content should be 'original', got %q",
						content,
					)
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

				var content string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content FROM messages WHERE id = 'm3'",
				).Scan(&content); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "original content" {
					t.Fatalf(
						"COALESCE should preserve original content when new is empty, got %q",
						content,
					)
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

				var content string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content FROM messages WHERE id = 'm4'",
				).Scan(&content); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "new content" {
					t.Fatalf("non-empty content should update, got %q", content)
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

				var content string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content FROM messages WHERE id = 'm5'",
				).Scan(&content); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "untouched" {
					t.Fatalf("empty setExprs should do nothing, got %q", content)
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

				var content string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content FROM messages WHERE id = 'm6'",
				).Scan(&content); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "hello world" {
					t.Fatalf(
						"bound-arg expression should append ' world' to 'hello', got %q",
						content,
					)
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

				var content, author, channel string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content, author_id, channel_id FROM messages WHERE id = 'm7'",
				).Scan(&content, &author, &channel); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "fresh-insert" {
					t.Fatalf("fresh insert should write content, got %q", content)
				}
				if author != "u1" {
					t.Fatalf(
						"fresh insert should write author_id even though not in updateCols, got %q",
						author,
					)
				}
				if channel != "c1" {
					t.Fatalf("fresh insert should write channel_id, got %q", channel)
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

				var content string
				if err := db.QueryRowContext(
					ctx,
					"SELECT content FROM messages WHERE id = 'm8'",
				).Scan(&content); err != nil {
					t.Fatalf("query: %v", err)
				}

				if content != "fresh-expr-insert" {
					t.Fatalf(
						"fresh insert should write provided content, not apply SetExpr, got %q",
						content,
					)
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
