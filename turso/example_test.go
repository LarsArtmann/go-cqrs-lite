package turso_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/turso/v2"
)

func ExampleOpenInMemory() {
	db, err := turso.OpenInMemory()
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer db.Close()

	fmt.Println(db != nil)

	// Output:
	// true
}

func ExampleNewEventStore() {
	db, _ := turso.OpenInMemory()
	defer db.Close()

	store, err := turso.NewEventStore(db)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	defer store.Close()

	fmt.Println(store != nil)

	// Output:
	// true
}
