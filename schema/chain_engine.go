package schema

import (
	"strconv"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// decodeOp is implemented by every op that decodes the payload into a field
// map and therefore owns a DecodeErrorPolicy.
type decodeOp interface {
	policy() DecodeErrorPolicy
}

func (o *fieldOp) policy() DecodeErrorPolicy     { return o.decodePolicy }
func (o *transformOp) policy() DecodeErrorPolicy { return o.decodePolicy }
func (o *splitOp) policy() DecodeErrorPolicy     { return o.decodePolicy }

type opOutcome int

const (
	opContinue opOutcome = iota
	opDone
	opDrop
	opSplit
)

func (c *Chain) match(evt event.Event) Op {
	if op, ok := c.exact[chainKey{evt.Type(), evt.SchemaVersion()}]; ok {
		return op
	}

	return c.byType[evt.Type()]
}

func (c *Chain) upcastAll(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, 0, len(events))

	for _, evt := range events {
		upcasted, err := c.upcastOne(evt, c.maxHops)
		if err != nil {
			return nil, errorfamily.WrapCorruption(
				err, "schema.chain_upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}

		result = append(result, upcasted...)
	}

	return result, nil
}

func (c *Chain) upcastOne(evt event.Event, budget int) ([]event.Event, error) {
	for {
		if budget < 0 {
			return nil, errorfamily.WrapCorruption(
				ErrChainCycle, "schema.chain_cycle",
				"upcast chain exceeded "+strconv.Itoa(c.maxHops)+" hops at "+string(evt.Type()),
			)
		}

		op := c.match(evt)
		if op == nil {
			return []event.Event{evt}, nil
		}

		results, outcome, err := applyOp(op, evt)
		if err != nil {
			return nil, err
		}

		switch outcome {
		case opContinue:
			evt = results[0]
			budget--
		case opDone:
			return []event.Event{evt}, nil
		case opDrop:
			return nil, nil
		case opSplit:
			return c.expandSplit(results, budget)
		}
	}
}

func (c *Chain) expandSplit(outputs []event.Event, budget int) ([]event.Event, error) {
	result := make([]event.Event, 0, len(outputs))

	for _, output := range outputs {
		upcasted, err := c.upcastOne(output, budget-1)
		if err != nil {
			return nil, err
		}

		result = append(result, upcasted...)
	}

	return result, nil
}

func applyOp(op Op, evt event.Event) ([]event.Event, opOutcome, error) {
	switch typed := op.(type) {
	case *renameTypeOp:
		next, err := rebuild(evt, evt.Payload(), typed.target, evt.SchemaVersion())

		return []event.Event{next}, opContinue, err
	case *dropOp:
		// Type-only removal: no payload work, nothing to rebuild.

		return nil, opDrop, nil
	case *splitOp:
		outputs, err := applySplitOp(typed, evt)

		return outputs, opSplit, err
	}

	decode, ok := op.(decodeOp)
	if !ok {
		return nil, opContinue, invalidOpErr("op cannot apply (schema.Op is sealed)")
	}

	next, outcome, err := applyDecodeOp(decode, evt)

	return []event.Event{next}, outcome, err
}

// applyDecodeOp runs the 1:1 payload ops (field ops and Transform): decode
// the payload into a field map (policy governs decode failures), mutate,
// re-encode with the SAME encoding, rebuild with schema version + 1.
func applyDecodeOp(op decodeOp, evt event.Event) (event.Event, opOutcome, error) {
	fields, err := decodeFieldMap(evt)
	if err != nil {
		return handleDecodeError(evt, op.policy(), err)
	}

	switch typed := op.(type) {
	case *transformOp:
		fields, err = typed.transform(fields)
		if err != nil {
			return nil, opContinue, wrapTransformErr(evt, err)
		}
	case *fieldOp:
		applyFieldOp(typed, fields)
	}

	payload, err := encodeFieldMap(evt, fields)
	if err != nil {
		return nil, opContinue, err
	}

	next, err := rebuild(evt, payload, evt.Type(), evt.SchemaVersion().Increment())

	return next, opContinue, err
}

func applyFieldOp(op *fieldOp, fields map[string]any) {
	switch op.kind {
	case fieldRename:
		if value, ok := fields[op.field]; ok {
			delete(fields, op.field)
			fields[op.renamedTo] = value
		}
	case fieldAdd:
		if _, ok := fields[op.field]; !ok {
			fields[op.field] = op.defaultValue
		}
	case fieldRemove:
		delete(fields, op.field)
	}
}

func applySplitOp(op *splitOp, evt event.Event) ([]event.Event, error) {
	fields, err := decodeFieldMap(evt)
	if err != nil {
		switch op.policy() {
		case PassthroughOnDecodeError:
			return []event.Event{evt}, nil
		case DropOnDecodeError:
			return nil, nil
		case FailOnDecodeError:
			return nil, err
		}
	}

	result := make([]event.Event, 0, len(op.outputs))

	for _, output := range op.outputs {
		produced, err := produceSplitOutput(evt, fields, output, op.sourceVersion)
		if err != nil {
			return nil, err
		}

		result = append(result, produced)
	}

	return result, nil
}

func produceSplitOutput(
	evt event.Event,
	fields map[string]any,
	output SplitOutput,
	sourceVersion event.SchemaVersion,
) (event.Event, error) {
	payload, err := output.payload(evt, fields)
	if err != nil {
		return nil, wrapTransformErr(evt, err)
	}

	data, err := encodeValue(evt, payload)
	if err != nil {
		return nil, err
	}

	// Fresh event ID: N outputs cannot share one identity. Everything else
	// (stream, position, metadata, timestamp, encoding) is inherited.
	produced, err := event.New(
		output.eventType,
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		data,
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithMetadata(evt.Metadata()),
		event.WithEncoding(evt.Encoding()),
		event.WithSchemaVersion(sourceVersion.Increment()),
	)
	if err != nil {
		return nil, wrapRebuildErr(evt, output.eventType, err)
	}

	return produced, nil
}

func handleDecodeError(
	evt event.Event,
	policy DecodeErrorPolicy,
	err error,
) (event.Event, opOutcome, error) {
	switch policy {
	case PassthroughOnDecodeError:
		return evt, opDone, nil
	case DropOnDecodeError:
		return nil, opDrop, nil
	case FailOnDecodeError:
		return nil, opContinue, err
	}

	return nil, opContinue, wrapDecodeErr(evt, err)
}

func wrapDecodeErr(evt event.Event, err error) error {
	return errorfamily.WrapCorruption(
		err, "schema.op_decode_failed",
		"decode "+string(evt.Encoding())+" payload of "+string(evt.Type())+
			" v"+evt.SchemaVersion().String(),
	)
}

func wrapTransformErr(evt event.Event, err error) error {
	return errorfamily.WrapCorruption(
		err, "schema.op_transform_failed",
		"transform "+string(evt.Type())+" v"+evt.SchemaVersion().String(),
	)
}

// decodeFieldMap decodes the payload with the codec the event's Encoding()
// stamp selects — self-describing events, mixed JSON/CBOR streams included.
func decodeFieldMap(evt event.Event) (map[string]any, error) {
	codecFor, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	var fields map[string]any
	if err := codecFor.Decode(evt.Payload(), &fields); err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	if fields == nil {
		fields = make(map[string]any)
	}

	return fields, nil
}

func encodeFieldMap(evt event.Event, fields map[string]any) ([]byte, error) {
	codecFor, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	payload, err := codecFor.Encode(fields)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err, "schema.op_encode_failed",
			"encode payload of "+string(evt.Type()),
		)
	}

	return payload, nil
}

// rebuild constructs the upcasted event for 1:1 ops: fresh instance,
// preserved identity (event ID, stream, position, timestamp, metadata,
// encoding), new type and schema version. payload is already encoded bytes.
func rebuild(
	evt event.Event,
	payload []byte,
	eventType event.Type,
	schemaVersion event.SchemaVersion,
) (event.Event, error) {
	upcasted, err := event.New(
		eventType,
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		payload,
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithMetadata(evt.Metadata()),
		event.WithEncoding(evt.Encoding()),
		event.WithSchemaVersion(schemaVersion),
	)
	if err != nil {
		return nil, wrapRebuildErr(evt, eventType, err)
	}

	return upcasted, nil
}

func wrapRebuildErr(evt event.Event, eventType event.Type, err error) error {
	return errorfamily.WrapCorruption(
		err, "schema.rebuild_failed",
		"rebuild "+string(eventType)+" v"+evt.SchemaVersion().String()+
			" (source "+string(evt.Type())+")",
	)
}

// encodeValue encodes a Split payload function's return value with the
// codec matching the SOURCE event's encoding, so payload bytes and the
// encoding stamp always agree ([]byte payloads pass through untouched).
func encodeValue(evt event.Event, payload any) ([]byte, error) {
	if raw, isBytes := payload.([]byte); isBytes {
		return raw, nil
	}

	codecFor, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	encoded, err := codecFor.Encode(payload)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err, "schema.op_encode_failed",
			"encode payload of "+string(evt.Type()),
		)
	}

	return encoded, nil
}
