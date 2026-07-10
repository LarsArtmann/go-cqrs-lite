// Package metadata provides the shared tracing and custom-data containers
// used by event, command, and query metadata types.
//
// Before this package existed, [Tracing] and [CustomData] lived inside event/.
// Every module that needed them (command/, query/) had to import event/,
// creating a tight coupling that violated the four-tier module model
// (ADR-0046). The metadata/ module breaks that dependency: command/ and query/
// embed these types directly without pulling in the full event/ package.
//
// # Types
//
// [Tracing] holds the cross-cutting tracing identifiers (CorrelationID,
// CausationID, UserID, RequestID). When embedded anonymously in a struct,
// encoding/json promotes the fields to the parent level, preserving the
// existing JSON shape: {"correlationId": "...", ...}.
//
// [CustomData[K]] is the generic base for command.Metadata and query.Metadata.
// It embeds [Tracing] and adds a typed Custom map. The type parameter K is a
// named string type (the module's own MetadataKey), so each module's custom
// keys are type-safe and cannot be accidentally mixed.
//
// # Usage
//
//	import "github.com/larsartmann/go-cqrs-lite/metadata/v3"
//
//	type MyKey string
//
//	type MyMetadata struct {
//	    metadata.CustomData[MyKey]
//	    // additional module-specific fields...
//	}
//
// # References
//
//   - ADR-0031: Typed Metadata fields
//   - ADR-0046: Four-tier module model
package metadata
