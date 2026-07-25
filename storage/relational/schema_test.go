package relational

import (
	"strings"
	"testing"
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

			if strings.Contains(ddl, "DEFAULT DEFAULT") {
				t.Fatalf("double DEFAULT in DDL:\n%s", ddl)
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
