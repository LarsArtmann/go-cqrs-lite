package schema_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsid "github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

func mustExampleStreamID() cqrsid.StreamID {
	streamID, err := cqrsid.ParseStreamID("01HK1540X0841Y0A6BSX1VKR95")
	if err != nil {
		panic(err)
	}

	return streamID
}

// ExampleCompile builds a validated chain from named ops: a type rename plus
// a field rename migrating stored v1 events toward the declared v2 shape.
// Compile rejects invalid declarations (duplicates, cycles, empty types) at
// construction time, before any read runs.
func ExampleCompile() {
	_, err := schema.Compile(
		schema.RenameType("user.renamed_legacy", "user.profiled"),
		schema.RenameField("user.profiled", 1, "name", "displayName"),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("chain compiled")

	// A duplicate (type, version) registration fails instead of silently
	// shadowing:
	_, err = schema.Compile(
		schema.RenameField("user.profiled", 1, "name", "displayName"),
		schema.RemoveField("user.profiled", 1, "legacyFlag"),
	)
	fmt.Println(err != nil)

	// Output:
	// chain compiled
	// true
}

// ExampleChain_SourceTransform composes the compiled chain onto a journal
// read path: every stored event comes back at its CURRENT shape — old
// type names renamed, old payload versions upcasted — without rewriting
// anything that is stored.
func ExampleChain_SourceTransform() {
	chain, err := schema.Compile(
		schema.RenameType("user.renamed_legacy", "user.profiled"),
		schema.RenameField("user.profiled", 1, "name", "displayName"),
	)
	if err != nil {
		panic(err)
	}

	legacy, err := event.NewEvent(
		"user.renamed_legacy",
		mustExampleStreamID(),
		"User",
		1,
		[]byte(`{"name":"Lars"}`),
		event.WithSchemaVersion(1),
	)
	if err != nil {
		panic(err)
	}

	upcasted, err := chain.SourceTransform()([]event.Event{legacy})
	if err != nil {
		panic(err)
	}

	fmt.Println(upcasted[0].Type())
	fmt.Println(string(upcasted[0].Payload()))
	fmt.Println(upcasted[0].SchemaVersion())

	// Output:
	// user.profiled
	// {"displayName":"Lars"}
	// 2
}

// ExampleDeclare shows the declaration-set form system.DomainConfig.Schema
// consumes: every event type declared at its current version with the ops
// migrating older stored versions toward it, compiled into ONE chain.
func ExampleDeclare() {
	_, err := schema.Declare(
		schema.Event("user.renamed_legacy", 1,
			schema.RenameType("user.renamed_legacy", "user.profiled"),
		),
		schema.Event("user.profiled", 2,
			schema.RenameField("user.profiled", 1, "name", "displayName"),
		),
		schema.Event("user.deleted", 1),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("declared 2 event types")

	// Declaration-level validation: duplicate event types across the set
	// fail composition.
	_, err = schema.Declare(
		schema.Event("user.profiled", 2),
		schema.Event("user.profiled", 1),
	)
	fmt.Println(err != nil)

	// The compiled chain applies to journal reads exactly like Compile's:
	// pass chain.SourceTransform() to event.DecorateJournal.

	// Output:
	// declared 2 event types
	// true
}
