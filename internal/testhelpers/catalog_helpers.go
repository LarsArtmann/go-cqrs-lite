package testhelpers

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

// AddCommandWithSchema adds a command with a schema to the registry.
func AddCommandWithSchema(r *catalog.Registry, svcID, id, name, version string, schema *catalog.Schema) {
	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      id,
		Name:    name,
		Version: version,
		Schema:  schema,
	}
	r.AddCommand(svcID, msg)
}

// AddEventWithSummary adds an event with summary and direction to the registry.
func AddEventWithSummary(r *catalog.Registry, svcID, id, name, version, summary string, direction catalog.Direction) {
	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        id,
		Name:      name,
		Version:   version,
		Summary:   summary,
		Direction: direction,
	}
	r.AddEvent(svcID, msg)
}

// AddCommandWithExamples adds a command with examples to the registry.
func AddCommandWithExamples(r *catalog.Registry, svcID, id, name, version string, examples ...json.RawMessage) {
	msg := catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       id,
		Name:     name,
		Version:  version,
		Examples: examples,
	}
	r.AddCommand(svcID, msg)
}
