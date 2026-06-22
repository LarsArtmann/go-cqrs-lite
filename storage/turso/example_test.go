package turso_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
)

func ExampleOpenInMemory() {
	conn, err := turso.OpenTemp("")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = conn.Close() }()

	fmt.Println(conn != nil)

	// Output:
	// true
}

func ExampleOpen() {
	db, err := turso.Open(turso.DbPath(":memory:"))
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = db.Close() }()

	fmt.Println(db != nil)

	// Output:
	// true
}

func ExampleNewEventStore() {
	conn, _ := turso.OpenTemp("")
	defer func() { _ = conn.Close() }()

	store, err := turso.NewEventStore(conn)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = store.Close() }()

	fmt.Println(store != nil)

	// Output:
	// true
}

func ExampleNewCommandStore() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = turso.InitSchema(context.Background(), db)

	store, err := turso.NewCommandStore(db)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = store.Close() }()

	fmt.Println(store != nil)

	// Output:
	// true
}

func ExampleNewQueryStore() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = turso.InitSchema(context.Background(), db)

	store, err := turso.NewQueryStore(db)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = store.Close() }()

	fmt.Println(store != nil)

	// Output:
	// true
}

func ExampleNewBackend() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = turso.InitSchema(context.Background(), db)

	backend, err := turso.NewBackend(db)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = backend.Close() }()

	// EventStore is eager; the rest are lazy.
	eventStore := backend.EventStore()
	cmdStore, _ := backend.CommandStore()
	qStore, _ := backend.QueryStore()

	fmt.Println(eventStore != nil && cmdStore != nil && qStore != nil)

	// Output:
	// true
}

func ExampleConfigurePool() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	turso.ConfigurePool(db)

	// Embedded LibSQL serializes writes — pool is capped at 1.
	fmt.Println(db.Stats().MaxOpenConnections)

	// Output:
	// 1
}

func ExampleInitSchemaWithIndexesAndOptimizations() {
	ctx := context.Background()

	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	if err := turso.InitSchemaWithIndexesAndOptimizations(ctx, db); err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("ready")

	// Output:
	// ready
}

// ExampleOpenSync_phantomTypes demonstrates that OpenSync requires
// phantom-typed inputs (DbPath, RemoteURL, AuthToken) for compile-time
// safety. This example validates the rejection of in-memory databases
// with remote sync — it does not require network access.
func ExampleOpenSync_phantomTypes() {
	ctx := context.Background()

	_, err := turso.OpenSync(
		ctx,
		turso.DbPath(":memory:"),
		turso.RemoteURL("libsql://example.turso.io"),
		turso.AuthToken("token"),
	)
	fmt.Println(err != nil)

	// Output:
	// true
}
