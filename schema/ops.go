package schema

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Op is a single named, declarative schema-evolution step.
//
// Ops are values: declare them in any order — [Compile] validates the whole
// set and matching at read time is by specificity (exact event type + schema
// version first, then type-only ops), never by declaration order.
//
// Versioning rules (AxonIQ semantics, adopted by ADR-track proposal
// 2026-10-09): payload ops ([RenameField], [AddField], [RemoveField],
// [Transform], [Split]) advance the schema version by exactly one;
// [RenameType] changes identity but not version; [Drop] removes the event.
// Only RenameType changes the event type.
//
// The interface is sealed (unexported method) so the matching contract stays
// under library control.
type Op interface {
	op()
}

// DecodeErrorPolicy selects what a payload-decoding op does when the stored
// payload cannot be decoded into a field map (a scalar or array payload,
// corrupted bytes, or an encoding mismatch).
type DecodeErrorPolicy int

const (
	// FailOnDecodeError (default) fails the read with a Corruption error:
	// a payload the declaration cannot decode is drift an operator must see.
	FailOnDecodeError DecodeErrorPolicy = iota

	// PassthroughOnDecodeError stops the chain for that event and returns it
	// unchanged (original payload and schema version).
	PassthroughOnDecodeError

	// DropOnDecodeError removes the event from the read batch entirely.
	// Deliberate skipping of known-garbage events — explicit and auditable,
	// never the default.
	DropOnDecodeError
)

// OpOption refines a single op. Currently one option exists:
// [WithDecodePolicy].
type OpOption func(*opConfig)

type opConfig struct {
	decodePolicy DecodeErrorPolicy
}

// WithDecodePolicy overrides the [DecodeErrorPolicy] of a payload-decoding
// op. The default is [FailOnDecodeError].
func WithDecodePolicy(policy DecodeErrorPolicy) OpOption {
	return func(c *opConfig) { c.decodePolicy = policy }
}

// SplitOutput declares one event produced by a [Split] op.
type SplitOutput struct {
	eventType event.Type
	payload   func(event.Event, map[string]any) (any, error)
}

// Producing declares one output of a [Split]: the new event type and a
// payload function receiving the source event plus its decoded field map.
// The returned value is encoded with the SOURCE event's codec ([]byte
// payloads pass through untouched), so a JSON source splits into JSON
// events and a CBOR source into CBOR events.
func Producing(
	eventType event.Type,
	payload func(event.Event, map[string]any) (any, error),
) SplitOutput {
	return SplitOutput{eventType: eventType, payload: payload}
}

// RenameType changes the event type of every matching event, keeping schema
// version and payload. Subsequent ops declared for the NEW type continue to
// apply (rename, then reshape).
//
// Invalid parameters (empty type, from == target) are rejected by [Compile].
func RenameType(from, target event.Type) Op {
	return &renameTypeOp{from: from, target: target}
}

// RenameField renames a payload field. If the field is absent the op is a
// no-op on the payload; the schema version still advances (the op ran).
//
// Invalid parameters are rejected by [Compile].
func RenameField(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	from, to string,
	opts ...OpOption,
) Op {
	return &fieldOp{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		kind:          fieldRename,
		field:         from,
		renamedTo:     to,
		opConfig:      newOpConfig(opts),
	}
}

// AddField adds a payload field with a default value when it is absent.
// An existing field is never overwritten. Use [RemoveField] for the
// inverse, and prefer explicit Remove+Add over in-place mutation.
//
// Invalid parameters are rejected by [Compile].
func AddField(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	field string,
	defaultValue any,
	opts ...OpOption,
) Op {
	return &fieldOp{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		kind:          fieldAdd,
		field:         field,
		defaultValue:  defaultValue,
		opConfig:      newOpConfig(opts),
	}
}

// RemoveField deletes a payload field. Absent fields are a no-op; the
// schema version still advances.
//
// Invalid parameters are rejected by [Compile].
func RemoveField(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	field string,
	opts ...OpOption,
) Op {
	return &fieldOp{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		kind:          fieldRemove,
		field:         field,
		opConfig:      newOpConfig(opts),
	}
}

// Transform applies a domain function to the decoded payload fields — the
// only place hand-written logic belongs (reshapes, derived fields, splits of
// one flat field into nested objects).
//
// Invalid parameters are rejected by [Compile].
func Transform(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	transform func(map[string]any) (map[string]any, error),
	opts ...OpOption,
) Op {
	return &transformOp{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		transform:     transform,
		opConfig:      newOpConfig(opts),
	}
}

// Split replaces one event with N events, each declared by [Producing].
// Outputs get fresh event IDs and inherit stream identity, position,
// metadata, timestamp, encoding, and schema version + 1.
//
// Invalid parameters are rejected by [Compile].
func Split(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	outputs ...SplitOutput,
) Op {
	return &splitOp{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		outputs:       outputs,
		opConfig:      newOpConfig(nil),
	}
}

// Drop removes every event of the given type from read batches. Use it for
// deliberately retired event types whose facts no fold consumes — the
// explicit alternative to silently skipping payloads (rejected as a default
// policy; see the 2026-10-09 proposal §11).
//
// Invalid parameters are rejected by [Compile].
func Drop(sourceType event.Type) Op {
	return &dropOp{sourceType: sourceType}
}

func newOpConfig(opts []OpOption) opConfig {
	config := opConfig{decodePolicy: FailOnDecodeError}

	for _, opt := range opts {
		opt(&config)
	}

	return config
}

type renameTypeOp struct {
	from   event.Type
	target event.Type
}

func (*renameTypeOp) op() {}

type dropOp struct {
	sourceType event.Type
}

func (*dropOp) op() {}

type fieldOpKind int

const (
	fieldRename fieldOpKind = iota
	fieldAdd
	fieldRemove
)

type fieldOp struct {
	opConfig
	sourceType    event.Type
	sourceVersion event.SchemaVersion
	kind          fieldOpKind
	field         string
	renamedTo     string
	defaultValue  any
}

func (*fieldOp) op() {}

type transformOp struct {
	opConfig
	sourceType    event.Type
	sourceVersion event.SchemaVersion
	transform     func(map[string]any) (map[string]any, error)
}

func (*transformOp) op() {}

type splitOp struct {
	opConfig
	sourceType    event.Type
	sourceVersion event.SchemaVersion
	outputs       []SplitOutput
}

func (*splitOp) op() {}
