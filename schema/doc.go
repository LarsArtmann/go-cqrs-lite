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
//
// # Named ops (declarative upcasting)
//
// Instead of one hand-written closure per evolution step, declare the steps
// as ops and compile them once:
//
//	chain, err := schema.Compile(
//		schema.RenameField("user.created", 1, "name", "displayName"),
//		schema.AddField("user.created", 1, "country", "US"),
//		schema.Transform("balance.updated", 1, reshapeToMoney),
//		schema.Drop("audit.legacy_ping"),
//	)
//
//	versioned := event.DecorateStore(store, nil, chain.SourceTransform())
//
// Ops are order-independent (matching is by event type + schema version,
// most specific first); Compile rejects duplicates, ambiguous renames, and
// cycles at build time. Payload ops decode via the event's own Encoding()
// stamp and re-encode with the same codec — mixed JSON/CBOR journals work.
// RenameType, Split, and Drop change event identity or count, so they need
// the batch-level chain.SourceTransform(); the 1:1 ops also convert to
// classic Upcasters via chain.Upcasters().
package schema
