package turso_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/turso/v2"
)

func ExampleOpenInMemory() {
	conn, err := turso.OpenInMemory()
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer func() { _ = conn.Close() }()

	fmt.Println(conn != nil)

	// Output:
	// true
}

func ExampleNewEventStore() {
	conn, _ := turso.OpenInMemory()
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
