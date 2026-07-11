package memory_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	stackmemory "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// ExampleNew shows the canonical Bundle entry point. A single call wires every
// capability (event store+bus, command/query/snapshot/checkpoint stores, and a
// read-model backend); the application code is identical across presets — only
// the one-line constructor changes (memory here, sqlite/pebble/postgres there).
//
// The deployer chooses the preset; the app imports only readmodel + stack and
// never sees a storage driver.
func ExampleNew() {
	b, err := stackmemory.New()
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = b.Close() }()

	store, err := stack.ReadModel[todoView, todoKey](
		b, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[todoView, todoKey]("todos:"),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	ctx := context.Background()

	if err := store.Set(ctx, "1", &todoView{Title: "buy milk"}); err != nil {
		fmt.Println("error:", err)

		return
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(got.Title)
	// Output: buy milk
}
