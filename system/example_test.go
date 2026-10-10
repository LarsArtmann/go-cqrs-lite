package system_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type exampleUserCreated struct {
	Name string `json:"name"`
}

// ExampleSchemas shows the declaration form that feeds BOTH DomainConfig
// inputs from one list: the upcasting declarations (DomainConfig.Schema) and
// the derived typed projection decoder (DomainConfig.ProjectionTypeDecoder).
// Pass both to system.New — the declaration, not the wiring, stays the single
// source of truth.
func ExampleSchemas() {
	decls, err := system.Schemas().
		Event[exampleUserCreated]("user.created", 2,
		schema.RenameField("user.created", 1, "name", "displayName"),
	).
		Event[struct{}]("user.deleted", 1).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println(len(decls.Declarations()))
	fmt.Println(decls.TypeDecoder() != nil)

	// Duplicate event types fail at Build, before composition:
	_, err = system.Schemas().
		Event[exampleUserCreated]("user.created", 2).
		Event[struct{}]("user.created", 1).
		Build()
	fmt.Println(err != nil)

	// Output:
	// 2
	// true
	// true
}
