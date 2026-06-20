// Package schema provides schema evolution for event-sourced systems via upcasting.
//
// When event payloads change over time (new fields, renamed properties, structural changes),
// upcasters transform old events to the current schema on load — without modifying stored data.
//
// # Quick Start
//
//	upcaster, _ := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (event.Event, error) {
//	    return event.NewEvent(evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
//	        UpdatedPayload{NewField: "default"},
//	        event.WithSchemaVersion(2),
//	    )
//	})
//
//	versioned, _ := schema.NewVersionedStore(store, upcaster)
//	events, _ := versioned.Load(ctx, ref)
//
// # VersionedStore
//
// Wraps any event.Store and applies upcasters transparently on read.
// The upcaster chain is validated at construction — cycle detection prevents
// infinite loops from misconfigured version jumps.
package schema
