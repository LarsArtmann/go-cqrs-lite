package memory

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// New returns a fully-wired in-memory [stack.Bundle].
//
// Every capability is set: event store + bus, command store, query store,
// snapshot store, checkpoint store, and read-model backend. The stores use
// thread-safe in-memory implementations from [memory]. The event bus uses
// [watermill.EventBus] (GoChannel-backed) per ADR-0028.
//
// The returned Bundle owns all resources; [stack.Bundle.Close] releases them.
// Nothing is persistent — data is lost when the process exits.
//
// For a partial Bundle (e.g. events only), use [stack.New] with individual
// options:
//
//	b, _ := stack.New(
//	    stack.WithEventStore(memory.NewMemoryStore()),
//	    stack.WithReadModels(kv.NewMemStore()),
//	)
func New() (*stack.Bundle, error) {
	b, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithCommandStore(memory.NewMemoryCommandStore()),
		stack.WithQueryStore(memory.NewMemoryQueryStore()),
		stack.WithSnapshotStore(memory.NewMemorySnapshotStore()),
		stack.WithCheckpointStore(memory.NewMemoryCheckpointStore()),
		stack.WithReadModels(kv.NewMemStore()),
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "memory.wire_bundle",
			"wire bundle")
	}

	return b, nil
}
