package sqlite_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

func TestMultiDB_SeparateViewDB(t *testing.T) {
	t.Parallel()

	// Open with a separate view DB.
	bundle, err := sqlite.New(
		":memory:",
		sqlite.WithViewDB(":memory:"),
	)
	if err != nil {
		t.Fatalf("sqlite.New with ViewDB: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	// Verify the read model store works.
	type userView struct {
		Name string
	}

	store := kv.NewTypedStore[userView, testKey](bundle.ReadModels)

	ctx := context.Background()
	id := testKey("user-1")

	err = store.Set(ctx, id, &userView{Name: "Alice"})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if val.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", val.Name)
	}
}

func TestMultiDB_AllSeparate(t *testing.T) {
	t.Parallel()

	bundle, err := sqlite.New(
		":memory:",
		sqlite.WithEventDB(":memory:"),
		sqlite.WithQueryDB(":memory:"),
		sqlite.WithViewDB(":memory:"),
	)
	if err != nil {
		t.Fatalf("sqlite.New with all separate DBs: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	// Verify the bundle has all capabilities wired.
	if bundle.ReadModels == nil {
		t.Fatal("expected ReadModels to be set")
	}

	if bundle.EventSink == nil {
		t.Fatal("expected EventSink to be set")
	}
}

type testKey string

func (t testKey) String() string { return string(t) }

var _ stack.Bundle = stack.Bundle{}
