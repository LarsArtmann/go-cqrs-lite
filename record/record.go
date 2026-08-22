// Package record defines the shared Record and CommonMetadata types that serve
// as the structural base for both Commands and Events in a CQRS/Event-Sourcing
// system (ADR-0111).
//
// A Record is an append-only, immutable entry in a stream. Events are records
// of facts (post-decision, immutable truth). Commands are records of intent
// (pre-decision, may be rejected). Both share the same structural shape.
package record

import (
	"errors"
	"strings"
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

	// CausationID identifies what caused this record: a command ID, a timer
	// ID, or whatever the tracing chain carried — the kind is implied by the
	// ID format, never stated.
	//
	// Deprecated: removed in v5. Use Cause, which records the causer's kind
	// explicitly instead of leaving it implied. The AsRecord bridges populate
	// both fields until the v5 cut.
	CausationID string

	// Cause identifies what produced this record — a command, a timer, an
	// event, or an ID-only tracing chain — with the kind stated explicitly.
	// The zero value means no cause was recorded (direct user action; the
	// Actor covers the "who").
	Cause Cause

	// ActorID identifies who or what produced this record: a user ID, a service
	// name, a cron job identifier, or "system" for internal processes —
	// kind-discriminated actors serialized as "kind:raw", legacy user IDs as
	// bare strings. Consumers pay a parse tax to recover the kind.
	//
	// Deprecated: removed in v5. Use Actor, which carries the
	// kind-discriminated union structurally. The AsRecord bridges populate
	// both fields until the v5 cut.
	ActorID string

	// Actor identifies who or what produced this record — user, bot, system,
	// or service — with the kind explicit at the type level instead of
	// smuggled through a "kind:raw" string. The zero value means no actor
	// was recorded.
	Actor Actor

	// ClientCreatedAt is the client's clock at the moment of creation. This may
	// lie (clock skew, offline tampering). Use for offline-first conflict
	// resolution and UX ("you created this at...").
	//
	// Deprecated: removed in v5. Use Created, which distinguishes "not
	// recorded" from the epoch via an explicit presence flag. The AsRecord
	// bridges populate both fields until the v5 cut.
	ClientCreatedAt time.Time

	// Created is the client's clock at the moment of creation — the best
	// available creation timestamp, which may lie (clock skew, offline
	// tampering). Use for offline-first conflict resolution and UX ("you
	// created this at..."). The zero Stamp means no client clock was recorded.
	Created Stamp

	// ServerReceivedAt is the server clock when the record arrived. Trustworthy
	// for server-side ordering but not for client intent. Set before store.Save.
	//
	// Deprecated: removed in v5. Use Received, which distinguishes "not
	// recorded" from the epoch via an explicit presence flag.
	ServerReceivedAt time.Time

	// Received is the server clock when the record arrived — trustworthy for
	// server-side ordering but not for client intent. Stamped before
	// store.Save. The zero Stamp means the record has not been received yet.
	Received Stamp

	// ServerStoredAt is the database's acknowledgment timestamp. This is what
	// the DB told us, not necessarily what it did internally. Use for
	// audit trails and eventual-consistency reconciliation.
	//
	// Deprecated: removed in v5. Use Stored, which distinguishes "not
	// recorded" from the epoch via an explicit presence flag.
	ServerStoredAt time.Time

	// Stored is the database's acknowledgment timestamp — what the DB told
	// us, not necessarily what it did internally. Use for audit trails and
	// eventual-consistency reconciliation. The zero Stamp means the record has
	// not been stored yet.
	Stored Stamp

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

// ErrInvalidStreamRef is returned by Validate when a StreamRef is malformed.
var ErrInvalidStreamRef = errors.New(
	"invalid stream reference: must be \"Type/EntityID\" with non-empty components",
)

// NewStreamRef constructs a StreamRef from a stream type and entity ID.
// The streamType may be empty (for command/query records that store the
// type separately), but entityID must be non-empty.
//
// NOTE — v5 breaking change (ADR-0123 Phase 8): at v5 this constructor
// becomes NewStreamRef(streamType, entityID string) (StreamRef, error) and
// returns ErrInvalidStreamRef for an empty entityID at construction. Call
// [StreamRef.Validate] now to catch malformed refs before the cut; empty
// streamType stays legal. Not deprecated — the constructor survives v5.
func NewStreamRef(streamType, entityID string) StreamRef {
	return StreamRef(streamType + "/" + entityID)
}

// NewStreamRefOrZero constructs a StreamRef, returning the zero StreamRef
// when the result would be malformed (an empty entity ID — the only invalid
// form, since an empty stream type is legal for command/query records).
//
// It is the producer-side counterpart to the v5 validating constructor
// (ADR-0123 Phase 8): adapters that cannot return an error use it so a
// Record either carries a well-formed StreamRef or none at all — never a
// malformed ref that fails [StreamRef.Validate] far from its cause.
func NewStreamRefOrZero(streamType, entityID string) StreamRef {
	ref := NewStreamRef(streamType, entityID)
	if ref.Validate() != nil {
		return ""
	}

	return ref
}

// Validate returns an error if the StreamRef is malformed: either no '/'
// separator, or the entity ID (component after the first '/') is empty.
// A non-empty stream type is recommended but not required, since command
// and query records store the type separately in Record.StreamType.
func (s StreamRef) Validate() error {
	idx := strings.IndexByte(string(s), '/')
	if idx < 0 {
		return ErrInvalidStreamRef
	}

	if idx == len(s)-1 {
		return ErrInvalidStreamRef
	}

	return nil
}

// Split returns the stream type and entity ID components of the StreamRef.
// The stream type may be empty (for refs like "/entityID" — command and
// query records store the type separately in Record.StreamType).
// Returns ("", "") if the format is invalid: no '/' found, or the entity ID
// (component after the first '/') is empty.
func (s StreamRef) Split() (string, string) {
	idx := strings.IndexByte(string(s), '/')
	if idx < 0 || idx == len(s)-1 {
		return "", ""
	}

	return string(s[:idx]), string(s[idx+1:])
}
