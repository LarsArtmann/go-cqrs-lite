// Package schema provides schema evolution for event-sourced systems via upcasting.
//
// When event payloads change over time (new fields, renamed properties, structural changes),
// upcasters transform old events to the current schema on load — without modifying stored data.
//
// # Quick Start
//
//	upcaster, _ := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (event.Event, error) {
//	    return event.NewEvent(evt.Type(), evt.StreamID(), evt.StreamType(), evt.Version(),
//	        UpdatedPayload{NewField: "default"},
//	        event.WithSchemaVersion(2),
//	    )
//	})
//
//	versioned := event.DecorateStore(store, nil, schema.UpcastSourceTransform(upcaster))
//	events, _ := versioned.Load(ctx, ref)
//
// # Upcasting stores
//
// UpcastSourceTransform composes with event.DecorateStore and applies
// upcasters transparently on every read path. The upcaster chain is
// validated at construction — cycle detection prevents
// infinite loops from misconfigured version jumps.
package schema
