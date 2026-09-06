package relational

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	_ "modernc.org/sqlite"
)

func newInjectionStore(t *testing.T) *RelationalStore {
	t.Helper()
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	schema := discordSchema()

	if err := schema.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	store, err := NewRelationalStore(schema, db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	return store
}

// hostileColumns attempt to smuggle SQL syntax through column references.
// The schema-validated store must reject all of them before any SQL runs.
var hostileColumns = []string{
	"name; DROP TABLE guilds",
	"name)--",
	"1=1) OR (1",
	"id UNION SELECT name FROM guilds",
	"rowid--",
	"*",
	"name ASC, created_at",
	"n'a'me",
}

func TestRelationalStore_Query_RejectsHostileColumns(t *testing.T) {
	store := newInjectionStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		err := store.Query(ctx, "guilds", []string{"id"}, kv.ViewQuery{
			Conditions: []kv.Condition{{Column: column, Op: kv.OpEq, Value: "x"}},
		}, func(_ func(dest ...any) error) error { return nil })
		if err == nil {
			t.Errorf("Query condition column %q: want rejection, got nil", column)
		}

		err = store.Query(ctx, "guilds", []string{"id"}, kv.ViewQuery{
			OrderBy: column,
		}, func(_ func(dest ...any) error) error { return nil })
		if err == nil {
			t.Errorf("Query order column %q: want rejection, got nil", column)
		}

		err = store.Query(
			ctx,
			"guilds",
			[]string{column},
			kv.ViewQuery{},
			func(_ func(dest ...any) error) error {
				return nil
			},
		)
		if err == nil {
			t.Errorf("Query select column %q: want rejection, got nil", column)
		}
	}
}

func TestRelationalStore_RejectsUnknownTableAndOperator(t *testing.T) {
	store := newInjectionStore(t)
	ctx := context.Background()

	err := store.Query(ctx, "guilds; DROP TABLE guilds", []string{"id"}, kv.ViewQuery{}, nil)
	if err == nil {
		t.Fatal("Query on hostile table name: want rejection, got nil")
	}

	err = store.Query(ctx, "guilds", []string{"id"}, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "name", Op: "= 'x' OR 1=1 --", Value: "x"}},
	}, func(_ func(dest ...any) error) error { return nil })
	if err == nil {
		t.Fatal("Query with hostile operator: want rejection, got nil")
	}
}

func TestRelationalStore_Count_RejectsHostileInputs(t *testing.T) {
	store := newInjectionStore(t)
	ctx := context.Background()

	for _, column := range hostileColumns {
		if _, err := store.Count(ctx, "guilds", []kv.Condition{
			{Column: column, Op: kv.OpEq, Value: "x"},
		}); err == nil {
			t.Errorf("Count with column %q: want rejection, got nil", column)
		}
	}

	if _, err := store.Count(ctx, "guilds", []kv.Condition{
		{Column: "name", Op: "IS NULL; DROP TABLE guilds"},
	}); err == nil {
		t.Fatal("Count with hostile operator: want rejection, got nil")
	}
}

func TestRelationalStore_RejectionNamesOffender(t *testing.T) {
	store := newInjectionStore(t)
	ctx := context.Background()

	err := store.Query(ctx, "guilds", []string{"id"}, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "secret_column", Op: kv.OpEq, Value: 1}},
	}, func(_ func(dest ...any) error) error { return nil })
	if err == nil {
		t.Fatal("Query with undeclared column: want rejection")
	}

	if !strings.Contains(err.Error(), "secret_column") {
		t.Errorf("rejection should name the offending column, got: %v", err)
	}
}

func TestRelationalStore_LegitimateQueriesStillPass(t *testing.T) {
	store := newInjectionStore(t)
	ctx := context.Background()

	if _, err := store.db.ExecContext(ctx,
		"INSERT INTO guilds (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"g1", "Test Guild", "2026-01-01", "2026-01-01"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var names []string

	err := store.Query(ctx, "guilds", []string{"name"}, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "id", Op: kv.OpEq, Value: "g1"}},
		OrderBy:    "id",
	}, func(scan func(dest ...any) error) error {
		var name string
		if err := scan(&name); err != nil {
			return err
		}

		names = append(names, name)

		return nil
	})
	if err != nil {
		t.Fatalf("Query with legitimate inputs: %v", err)
	}

	if len(names) != 1 || names[0] != "Test Guild" {
		t.Fatalf("Query results: got %v, want [Test Guild]", names)
	}

	count, err := store.Count(ctx, "guilds", nil)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	if count != 1 {
		t.Fatalf("Count: got %d, want 1", count)
	}
}
