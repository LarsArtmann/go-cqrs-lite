// Package record defines the shared Record and CommonMetadata types that serve
// as the structural base for both Commands and Events in a CQRS/Event-Sourcing
// system (ADR-0111).
//
// A Record is an append-only, immutable entry in a stream. Events are records
// of facts (post-decision, immutable truth). Commands are records of intent
// (pre-decision, may be rejected). Both share the same structural shape.
package record

import (
	"fmt"
	"time"
)

// StreamRef identifies a stream as "StreamType/EntityID", e.g. "User/01J...".
// It is the primary key for event-sourced aggregates and command logs.
type StreamRef string

// String returns the stream reference as a string.
func (s StreamRef) String() string { return string(s) }

// CommonMetadata carries the metadata shared by all record types (events and
// commands). It replaces the parallel metadata hierarchies in event/ and
// command/ with a single, honest structure (ADR-0111).
type CommonMetadata struct {
	// CorrelationID links a chain of related records across the system.
	// A user action that triggers a command, which emits events, which trigger
	// sagas that emit more commands — all share one CorrelationID.
	CorrelationID string

	// CausationID identifies what caused this record. For events emitted by a
	// command, this is the command ID. For scheduled commands, the timer ID.
	// For direct user actions, this is empty — the ActorID covers the "who".
	CausationID string

	// ActorID identifies who or what produced this record: a user ID, a service
	// name, a cron job identifier, or "system" for internal processes.
	ActorID string

	// ClientCreatedAt is the client's clock at the moment of creation. This may
	// lie (clock skew, offline tampering). Use for offline-first conflict
	// resolution and UX ("you created this at...").
	ClientCreatedAt time.Time

	// ServerReceivedAt is the server clock when the record arrived. Trustworthy
	// for server-side ordering but not for client intent. Set before store.Save.
	ServerReceivedAt time.Time

	// ServerStoredAt is the database's acknowledgment timestamp. This is what
	// the DB told us, not necessarily what it did internally. Use for
	// audit trails and eventual-consistency reconciliation.
	ServerStoredAt time.Time

	// SchemaVersion is the payload schema version. Set once at record creation
	// and never changed. Enables upcasting: different versions of the same
	// logical event type can coexist in the stream.
	SchemaVersion int
}

// Record is the shared structural base for Commands and Events (ADR-0111).
// Both are append-only, immutable entries in streams. The only conceptual
// difference: Events are facts (post-decision), Commands are intents
// (pre-decision, may be rejected).
type Record struct {
	// Type is the domain event type ("user.created") or command type
	// ("create_user"). This drives fold dispatch in projections and deciders.
	Type string

	// Payload is the encoded payload bytes. The encoding is self-describing:
	// the record carries its codec stamp so mixed JSON+CBOR streams decode
	// correctly.
	Payload []byte

	// StreamID identifies the stream this record belongs to, e.g. "User/01J...".
	StreamID StreamRef

	// StreamType is the aggregate or category name: "User", "Order", "Command".
	// For events, this is the aggregate root. For commands, it is "Command" or
	// the target aggregate type.
	StreamType string

	// Version is the 1-indexed position of this record within its stream.
	// The first record in a stream has Version 1.
	Version int64

	// MetaData carries shared metadata (correlation, causation, timestamps).
	MetaData CommonMetadata
}

// NewStreamRef constructs a StreamRef from a stream type and entity ID.
func NewStreamRef(streamType, entityID string) StreamRef {
	return StreamRef(fmt.Sprintf("%s/%s", streamType, entityID))
}

// Split returns the stream type and entity ID components of the StreamRef.
// Returns ("", "") if the format is invalid.
func (s StreamRef) Split() (streamType string, entityID string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return string(s[:i]), string(s[i+1:])
		}
	}
	return "", ""
}
