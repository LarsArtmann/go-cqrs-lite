// Package metadata provides the shared custom-data container used by command
// and query metadata types.
//
// Before this package existed, the tracing and custom-data types lived inside
// event/. Every module that needed them (command/, query/) had to import
// event/, creating a tight coupling that violated the seven-tier module model
// (ADR-0046). The metadata/ module breaks that dependency: command/ and query/
// embed these types directly without pulling in the full event/ package.
//
// # Types
//
// [CustomData[K]] is the generic base for command.Metadata and query.Metadata.
// It embeds [record.CommonMetadata] (the shared tracing fields — CorrelationID,
// CausationID, ActorID, RequestID — defined in the record/ package since
// ADR-0111 Phase 3) and adds a typed Custom map. The type parameter K is a
// named string type (the module's own MetadataKey), so each module's custom
// keys are type-safe and cannot be accidentally mixed.
//
// CustomData is deprecated: model metadata as a standalone struct embedding
// [record.CommonMetadata] directly instead. See command.Metadata and
// query.Metadata for the preferred pattern.
//
// # Usage (deprecated pattern)
//
//	import "github.com/larsartmann/go-cqrs-lite/metadata/v4"
//
//	type MyKey string
//
//	type MyMetadata struct {
//	    metadata.CustomData[MyKey]
//	    // additional module-specific fields...
//	}
//
// # Preferred pattern (ADR-0111)
//
//	import "github.com/larsartmann/go-cqrs-lite/record/v4"
//
//	type MyMetadata struct {
//	    record.CommonMetadata
//	    // additional module-specific fields...
//	}
//
// # References
//
//   - ADR-0031: Typed Metadata fields
//   - ADR-0046: Seven-tier module model
//   - ADR-0111: Record type extraction (Phase 3 moved tracing to record.CommonMetadata)
package metadata
