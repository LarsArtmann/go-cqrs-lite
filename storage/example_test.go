package storage_test

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"

	_ "modernc.org/sqlite"
)

func ExampleNewSQLiteEventStore() {
	ctx := context.Background()

	db, _ := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	defer db.Close()

	_ = storage.SQLiteInitSchema(ctx, db)

	store, _ := storage.NewSQLiteEventStore(db)

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, _ := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"Alice"}`))

	_ = store.Save(ctx, ref, []event.Event{evt}, 0)

	events, _ := store.Load(ctx, ref)
	fmt.Println(len(events))
	fmt.Println(events[0].Type())

	// Output:
	// 1
	// user.created
}
