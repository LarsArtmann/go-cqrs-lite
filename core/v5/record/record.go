// Package record defines the shared Record and CommonMetadata types that serve
// as the structural base for both Commands and Events in a CQRS/Event-Sourcing
// system (ADR-0111).
//
// A Record is an append-only, immutable entry in a stream. Events are records
// of facts (post-decision, immutable truth). Commands are records of intent
// (pre-decision, may be rejected). Both share the same structural shape.
package record

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
)

// StreamRef identifies a stream by its type and instance ID. It is an
// alias of [id.StreamRef] (ADR-0166): one canonical struct, one validation
// rule set (empty type ILLEGAL, zero ID ILLEGAL — enforced at construction
// by [id.NewStreamRef]), one storage form ([id.StreamRef.StreamKey],
// "type:bare-id"). The zero StreamRef means "not a member of any stream":
// commands and queries that carry no stream identity use it instead of a
// fabricated ref.
type StreamRef = id.StreamRef

// CommonMetadata carries the metadata shared by all record types (events and
// commands). It replaces the parallel metadata hierarchies in event/ and
// command/ with a single, honest structure (ADR-0111).
type CommonMetadata struct {
	// CorrelationID links a chain of related records across the system.
	// A user action that triggers a command, which emits events, which trigger
	// sagas that emit more commands — all share one CorrelationID.
	CorrelationID CorrelationID

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
	// ID is the record instance's unique identifier: the EventID for events,
	// the CommandID for commands, the RequestID for queries. It survived the
	// AsRecord bridges before this field existed — identity was dropped on
	// the floor (review P5). The zero value means the source carried no
	// instance ID; construct through [ParseInstanceID] — constructors are the
	// only doors in.
	ID InstanceID

	// Type is the domain event type ("user.created") or command type
	// ("create_user"). This drives fold dispatch in projections and deciders.
	// event.Type, command.Type, and query.Type are aliases of this type
	// (ADR-0111): one definition, so the per-module copies cannot drift.
	Type Type

	// Payload is the encoded payload fused with its codec stamp (see the
	// [Payload] state table): absent (zero), opaque (no stamp at this
	// layer), envelope-wrapped, or self-describing (JSON/CBOR). One fact,
	// one field — presence never rides on a separate stamp's zero value.
	Payload Payload

	// Stream identifies the stream this record belongs to: {Type, ID} at full
	// fidelity. The zero value means the record is not a stream member
	// (queries; commands without a target stream) — never a malformed ref
	// (ADR-0166). Construct through [id.NewStreamRef], or
	// [id.NewStreamRefOrZero] where no error can surface; parse storage keys
	// with [id.ParseStreamRef].
	Stream StreamRef

	// Version is the 1-indexed position of this record within its stream.
	// The first record in a stream has Version 1. The zero value means
	// "unversioned" — the honest state for commands and queries, which carry
	// no stream position. Negative positions are unrepresentable.
	Version Version

	// MetaData carries shared metadata (correlation, causation, timestamps).
	MetaData CommonMetadata
}
