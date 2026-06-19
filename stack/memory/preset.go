package memory

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
)

// New returns a fully-wired in-memory [stack.Bundle].
//
// Every capability is set: event store + bus, command store, query store,
// snapshot store, checkpoint store, and read-model backend. All backed by
// thread-safe in-memory implementations from the [memory] and [kv] packages.
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
		stack.WithBus(memory.NewMemoryBus()),
		stack.WithCommandStore(memory.NewMemoryCommandStore()),
		stack.WithQueryStore(memory.NewMemoryQueryStore()),
		stack.WithSnapshotStore(memory.NewMemorySnapshotStore()),
		stack.WithCheckpointStore(memory.NewMemoryCheckpointStore()),
		stack.WithReadModels(kv.NewMemStore()),
	)
	if err != nil {
		return nil, fmt.Errorf("stack/memory: wire bundle: %w", err)
	}

	return b, nil
}
