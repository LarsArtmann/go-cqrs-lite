package relational

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRelationalColumn_Default(t *testing.T) {
	tests := []struct {
		name     string
		col      RelationalColumn
		wantExpr string
	}{
		{
			name:     "integer default zero",
			col:      RelationalColumn{Name: "count", Type: "INTEGER", Default: "0"},
			wantExpr: "count INTEGER NOT NULL DEFAULT 0",
		},
		{
			name:     "text default empty string",
			col:      RelationalColumn{Name: "label", Type: "TEXT", Default: "''"},
			wantExpr: "label TEXT NOT NULL DEFAULT ''",
		},
		{
			name: "current timestamp",
			col: RelationalColumn{
				Name:    "created_at",
				Type:    "TEXT",
				Default: "CURRENT_TIMESTAMP",
			},
			wantExpr: "created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP",
		},
		{
			name:     "boolean default",
			col:      RelationalColumn{Name: "active", Type: "INTEGER", Default: "1"},
			wantExpr: "active INTEGER NOT NULL DEFAULT 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := RelationalTable{Name: "test", Columns: []RelationalColumn{tt.col}}
			ddl := tbl.DDL()

			if !strings.Contains(ddl, tt.wantExpr) {
				t.Fatalf("DDL missing %q:\n%s", tt.wantExpr, ddl)
			}

			if strings.Count(ddl, "DEFAULT") > 1 {
				t.Fatalf("multiple DEFAULT clauses in DDL:\n%s", ddl)
			}
		})
	}
}

func TestRelationalColumn_DefaultEmpty(t *testing.T) {
	tbl := RelationalTable{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
		},
	}

	ddl := tbl.DDL()

	if strings.Contains(ddl, "DEFAULT") {
		t.Fatalf("empty Default should not emit DEFAULT clause:\n%s", ddl)
	}
}

func TestRelationalColumn_References(t *testing.T) {
	tests := []struct {
		name     string
		col      RelationalColumn
		wantExpr string
	}{
		{
			name:     "simple FK",
			col:      RelationalColumn{Name: "guild_id", Type: "TEXT", References: "guilds(id)"},
			wantExpr: "guild_id TEXT NOT NULL REFERENCES guilds(id)",
		},
		{
			name: "FK with cascade",
			col: RelationalColumn{
				Name:       "parent_id",
				Type:       "TEXT",
				References: "parents(id) ON DELETE CASCADE",
			},
			wantExpr: "parent_id TEXT NOT NULL REFERENCES parents(id) ON DELETE CASCADE",
		},
		{
			name: "nullable FK",
			col: RelationalColumn{
				Name:       "channel_id",
				Type:       "TEXT",
				Nullable:   true,
				References: "channels(id)",
			},
			wantExpr: "channel_id TEXT REFERENCES channels(id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := RelationalTable{Name: "test", Columns: []RelationalColumn{tt.col}}
			ddl := tbl.DDL()

			if !strings.Contains(ddl, tt.wantExpr) {
				t.Fatalf("DDL missing %q:\n%s", tt.wantExpr, ddl)
			}
		})
	}
}

func TestRelationalColumn_ReferencesEmpty(t *testing.T) {
	tbl := RelationalTable{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
		},
	}

	ddl := tbl.DDL()

	if strings.Contains(ddl, "REFERENCES") {
		t.Fatalf("empty References should not emit REFERENCES clause:\n%s", ddl)
	}
}

func TestRelationalColumn_Unique(t *testing.T) {
	tbl := RelationalTable{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "email", Type: "TEXT", Unique: true},
		},
	}

	ddl := tbl.DDL()

	if !strings.Contains(ddl, "email TEXT NOT NULL UNIQUE") {
		t.Fatalf("DDL missing UNIQUE:\n%s", ddl)
	}
}

func TestRelationalColumn_UniqueFalse(t *testing.T) {
	tbl := RelationalTable{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "name", Type: "TEXT", Unique: false},
		},
	}

	ddl := tbl.DDL()

	if strings.Contains(ddl, "UNIQUE") {
		t.Fatalf("Unique=false should not emit UNIQUE:\n%s", ddl)
	}
}

func TestRelationalColumn_CombinedDefaultReferencesNullable(t *testing.T) {
	tbl := RelationalTable{
		Name:       "messages",
		PrimaryKey: []string{"id"},
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
			{Name: "guild_id", Type: "TEXT", Nullable: true, References: "guilds(id)"},
			{Name: "status", Type: "TEXT", Default: "'pending'"},
			{Name: "created_at", Type: "TEXT", Default: "CURRENT_TIMESTAMP"},
		},
	}

	ddl := tbl.DDL()

	wantParts := []string{
		"id TEXT NOT NULL",
		"guild_id TEXT REFERENCES guilds(id)",
		"status TEXT NOT NULL DEFAULT 'pending'",
		"created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP",
	}

	for _, want := range wantParts {
		if !strings.Contains(ddl, want) {
			t.Fatalf("DDL missing %q:\n%s", want, ddl)
		}
	}
}

func TestRelationalColumn_UniqueAndPrimaryKey(t *testing.T) {
	tbl := RelationalTable{
		Name:       "test",
		PrimaryKey: []string{"id"},
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT", Unique: true},
		},
	}

	ddl := tbl.DDL()

	// Both UNIQUE and PRIMARY KEY should be emitted — the DB engine handles the
	// redundancy (PK implies uniqueness, but explicit UNIQUE is harmless and
	// signals intent for auto-migration tooling).
	if !strings.Contains(ddl, "UNIQUE") {
		t.Fatalf("DDL should contain UNIQUE even on PK column:\n%s", ddl)
	}

	if !strings.Contains(ddl, "PRIMARY KEY (id)") {
		t.Fatalf("DDL should contain PRIMARY KEY:\n%s", ddl)
	}
}

func TestRelationalSchema_ValidateAcceptsNewFields(t *testing.T) {
	schema := RelationalSchema{Tables: []RelationalTable{
		{
			Name:       "guilds",
			PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "name", Type: "TEXT", Default: "'unknown'"},
			},
		},
		{
			Name:       "channels",
			PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "guild_id", Type: "TEXT", References: "guilds(id)"},
				{Name: "token", Type: "TEXT", Unique: true},
			},
		},
	}}

	if err := schema.Validate(); err != nil {
		t.Fatalf("Validate should accept Default/References/Unique: %v", err)
	}
}

func TestRelationalTable_IndexSpec_DDL(t *testing.T) {
	tbl := RelationalTable{
		Name:       "messages",
		PrimaryKey: []string{"id"},
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
			{Name: "channel_id", Type: "TEXT"},
			{Name: "created_at", Type: "TEXT"},
			{Name: "deleted_at", Type: "TEXT", Nullable: true},
		},
		Indexes: []IndexSpec{
			{Name: "idx_messages_channel", Columns: []string{"channel_id"}},
			{Name: "idx_messages_created", Columns: []string{"created_at"}},
			{
				Name:    "idx_messages_not_deleted",
				Columns: []string{"channel_id", "created_at"},
				Where:   "deleted_at IS NULL",
			},
		},
	}

	schema := RelationalSchema{Tables: []RelationalTable{tbl}}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := schema.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, idx := range tbl.Indexes {
		var name string
		err := db.QueryRowContext(
			context.Background(),
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
			idx.Name,
		).Scan(&name)
		if err != nil {
			t.Fatalf("index %q not created: %v", idx.Name, err)
		}
	}
}

func TestRelationalTable_UniqueSpec_DDL(t *testing.T) {
	tbl := RelationalTable{
		Name:       "reactions",
		PrimaryKey: []string{"id"},
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
			{Name: "message_id", Type: "TEXT"},
			{Name: "user_id", Type: "TEXT"},
			{Name: "emoji", Type: "TEXT"},
		},
		Uniques: []UniqueSpec{
			{Name: "uq_reaction_user_emoji", Columns: []string{"message_id", "user_id", "emoji"}},
		},
	}

	ddl := tbl.DDL()

	if !strings.Contains(ddl, "UNIQUE (message_id, user_id, emoji)") {
		t.Fatalf("DDL missing composite UNIQUE:\n%s", ddl)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.ExecContext(context.Background(), tbl.DDL()); err != nil {
		t.Fatalf("create table with composite unique: %v", err)
	}

	_, err = db.ExecContext(
		context.Background(),
		"INSERT INTO reactions (id, message_id, user_id, emoji) VALUES ('1', 'm1', 'u1', '👍')",
	)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = db.ExecContext(
		context.Background(),
		"INSERT INTO reactions (id, message_id, user_id, emoji) VALUES ('2', 'm1', 'u1', '👍')",
	)
	if err == nil {
		t.Fatalf("duplicate insert should violate composite unique constraint")
	}
}

func TestRelationalSchema_Validate_RejectsUnknownIndexColumn(t *testing.T) {
	schema := RelationalSchema{Tables: []RelationalTable{{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
		},
		Indexes: []IndexSpec{
			{Name: "idx_bogus", Columns: []string{"nonexistent"}},
		},
	}}}

	err := schema.Validate()
	if err == nil {
		t.Fatalf("Validate should reject unknown index column")
	}

	if !strings.Contains(err.Error(), "nonexistent") {
		t.Fatalf("error should name the unknown column: %v", err)
	}
}

func TestRelationalSchema_Validate_RejectsUnknownUniqueColumn(t *testing.T) {
	schema := RelationalSchema{Tables: []RelationalTable{{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
		},
		Uniques: []UniqueSpec{
			{Name: "uq_bogus", Columns: []string{"ghost_col"}},
		},
	}}}

	err := schema.Validate()
	if err == nil {
		t.Fatalf("Validate should reject unknown unique column")
	}

	if !strings.Contains(err.Error(), "ghost_col") {
		t.Fatalf("error should name the unknown column: %v", err)
	}
}

func TestRelationalSchema_Validate_RejectsIndexNoName(t *testing.T) {
	schema := RelationalSchema{Tables: []RelationalTable{{
		Name: "test",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
		},
		Indexes: []IndexSpec{
			{Columns: []string{"id"}},
		},
	}}}

	err := schema.Validate()
	if err == nil {
		t.Fatalf("Validate should reject index without name")
	}
}

func TestRelationalSchema_Migrate_CreatesIndexesAfterTables(t *testing.T) {
	schema := RelationalSchema{Tables: []RelationalTable{
		{
			Name:       "guilds",
			PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "name", Type: "TEXT"},
			},
		},
		{
			Name:       "channels",
			PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "guild_id", Type: "TEXT", References: "guilds(id)"},
			},
			Indexes: []IndexSpec{
				{Name: "idx_channels_guild", Columns: []string{"guild_id"}},
			},
		},
	}}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := schema.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var count int
	err = db.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_channels_guild'",
	).Scan(&count)
	if err != nil {
		t.Fatalf("query index: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected index idx_channels_guild to exist, got count=%d", count)
	}

	if err := schema.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate should be idempotent: %v", err)
	}
}

// TestSchema_ValidateAcceptsOrderedIndexColumns verifies that an IndexSpec may
// declare columns with trailing ASC/DESC sort qualifiers (e.g. "created_at
// DESC" in a composite index) and still pass Validate + Migrate. The qualifier
// must be stripped before checking the column name against the table's columns.
func TestSchema_ValidateAcceptsOrderedIndexColumns(t *testing.T) {
	t.Parallel()

	schema := RelationalSchema{Tables: []RelationalTable{{
		Name: "events",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
			{Name: "channel_id", Type: "TEXT"},
			{Name: "created_at", Type: "DATETIME"},
		},
		PrimaryKey: []string{"id"},
		Indexes: []IndexSpec{
			{Name: "idx_channel_created_desc", Columns: []string{"channel_id", "created_at DESC", "id DESC"}},
			{Name: "idx_created_asc", Columns: []string{"created_at ASC"}},
			{Name: "idx_plain", Columns: []string{"channel_id"}},
		},
	}}}

	if err := schema.Validate(); err != nil {
		t.Fatalf("Validate should accept ordered index columns: %v", err)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := schema.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate should create ordered indexes: %v", err)
	}

	var sql string
	err = db.QueryRowContext(
		context.Background(),
		"SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_channel_created_desc'",
	).Scan(&sql)
	if err != nil {
		t.Fatalf("query index sql: %v", err)
	}

	want := "CREATE INDEX idx_channel_created_desc ON events (channel_id, created_at DESC, id DESC)"
	if sql != want {
		t.Fatalf("index SQL mismatch:\ngot:  %s\nwant: %s", sql, want)
	}
}

// TestSchema_ValidateRejectsTrulyUnknownIndexColumn verifies that the ASC/DESC
// stripping does not mask a genuinely unknown column name.
func TestSchema_ValidateRejectsTrulyUnknownIndexColumn(t *testing.T) {
	t.Parallel()

	schema := RelationalSchema{Tables: []RelationalTable{{
		Name: "events",
		Columns: []RelationalColumn{
			{Name: "id", Type: "TEXT"},
		},
		PrimaryKey: []string{"id"},
		Indexes: []IndexSpec{
			{Name: "idx_bad", Columns: []string{"nonexistent DESC"}},
		},
	}}}

	err := schema.Validate()
	if err == nil {
		t.Fatal("Validate should reject an index column that does not exist after stripping the qualifier")
	}
}
