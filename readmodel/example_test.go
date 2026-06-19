package readmodel_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
)

// exampleTodoID stands in for a branded id.Of[todoMarker]: a named type whose
// String() method yields its value. The real id.Of satisfies the same
// fmt.Stringer constraint, so readmodel.Store works with it unchanged.
type exampleTodoID string

func (id exampleTodoID) String() string { return string(id) }

type exampleTodo struct {
	Title string `json:"title"`
}

// This example shows the primary intended use: a typed read-model store
// addressed by a branded identifier, over a kv.Backend chosen by the deployer.
//
// The deployer picks the Backend (here kv.MemStore for a test; in production
// pebble.KVAdapter or a SQL-backed KV store). The application code is
// identical in every deployment.
func ExampleNew() {
	// Deployer chooses the Backend. Application sees only readmodel.
	backend := kv.NewMemStore()

	store := readmodel.New[exampleTodo, exampleTodoID](
		backend,
		readmodel.WithKeyPrefix[exampleTodo, exampleTodoID]("todos:"),
	)

	ctx := context.Background()
	id := exampleTodoID("01H8XGJBY")

	if err := store.Set(ctx, id, &exampleTodo{Title: "ship the bundle layer"}); err != nil {
		fmt.Println("Set:", err)
		return
	}

	got, err := store.Get(ctx, id)
	if err != nil {
		fmt.Println("Get:", err)
		return
	}

	fmt.Println(got.Title)
	// Output: ship the bundle layer
}
