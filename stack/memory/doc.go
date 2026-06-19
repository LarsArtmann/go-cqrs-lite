// Package memory provides an all-in-memory preset for [stack.Bundle].
//
// It is the fastest way to get a working Bundle for development, testing, and
// prototyping: every capability (event store, bus, command store, query store,
// snapshot store, checkpoint store, and read-model backend) is wired from the
// existing [memory] and [kv] packages — no new implementation, just assembly.
//
//	b, err := memory.New()
//	defer b.Close()
//
//	repo, _ := stack.Repository[State](b, decider)
//	store, _ := stack.ReadModel[Todo, TodoID](b, codec.JSONCodec{})
//
// Nothing in this preset is persistent: all data is lost when the process
// exits. For persistence, use [github.com/larsartmann/go-cqrs-lite/stack/sqlite/v2]
// or [github.com/larsartmann/go-cqrs-lite/stack/pebble/v2].
package memory
