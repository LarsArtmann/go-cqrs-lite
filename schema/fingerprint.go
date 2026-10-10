package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// FingerprintMetadataKey is the event metadata key the opt-in write-path
// stamp uses (see [FingerprintStamp]).
const FingerprintMetadataKey = "schema.fingerprint"

// Fingerprint returns a stable identity of the declaration: the event type,
// the current version, and the sorted migration steps — hashed with SHA-256
// (an integrity fingerprint, never a cipher; no secret is involved or
// recoverable). Two declarations with the same shape fingerprint decode
// identically; any op added, removed, or re-versioned changes it, and so does
// a current-version bump.
//
// Known limit, by design: a [Transform] op contributes its PRESENCE (type,
// source version) but not its closure body — Go function values are not
// hashable. The fingerprint pins the migration ladder's shape, not its code;
// changing only a transform's internals does not change the fingerprint.
func (s EventSchema) Fingerprint() string {
	steps := make([]string, 0, len(s.ops))

	for _, op := range s.ops {
		steps = append(steps, opFingerprintStep(op))
	}

	sort.Strings(steps)

	sum := sha256.Sum256([]byte(
		fmt.Sprintf("v1|%s|%d|%s", s.eventType, s.currentVersion, fmt.Sprintf("%v", steps)),
	))

	return "v1:" + hex.EncodeToString(sum[:8])
}

// opFingerprintStep derives one op's descriptor. Field ops contribute their
// kind, field names, and default value; renames contribute both sides; the
// closure-carrying ops (Transform, Split) contribute kind + arity only.
func opFingerprintStep(op Op) string {
	switch typed := op.(type) {
	case *renameTypeOp:
		return fmt.Sprintf("rename:%s->%s", typed.from, typed.target)
	case *fieldOp:
		switch typed.kind {
		case fieldRename:
			return fmt.Sprintf(
				"field-rename:%s@%d:%s->%s",
				typed.sourceType,
				typed.sourceVersion,
				typed.field,
				typed.renamedTo,
			)
		case fieldAdd:
			return fmt.Sprintf(
				"field-add:%s@%d:%s=%v",
				typed.sourceType,
				typed.sourceVersion,
				typed.field,
				typed.defaultValue,
			)
		case fieldRemove:
			return fmt.Sprintf(
				"field-remove:%s@%d:%s",
				typed.sourceType,
				typed.sourceVersion,
				typed.field,
			)
		}
	case *transformOp:
		return fmt.Sprintf("transform:%s@%d", typed.sourceType, typed.sourceVersion)
	case *splitOp:
		return fmt.Sprintf(
			"split:%s@%d:%d-outputs",
			typed.sourceType,
			typed.sourceVersion,
			len(typed.outputs),
		)
	case *dropOp:
		return fmt.Sprintf("drop:%s", typed.sourceType)
	}

	return fmt.Sprintf("unknown:%T", op)
}

// FingerprintStamp returns an [event.SinkTransform] that stamps every written
// event whose type is declared with that declaration's [EventSchema.Fingerprint]
// under [FingerprintMetadataKey] (undeclared types pass through untouched).
// Compose it on the WRITE path — `event.DecorateStore(store, schema.
// FingerprintStamp(declared, schema.FingerprintMetadataKey), nil)` — so the
// durable journal records which declared shape produced each event. Events
// already carrying the stamp are left as-is (idempotent on re-writes).
func FingerprintStamp(declared []EventSchema, metadataKey string) event.SinkTransform {
	if metadataKey == "" {
		metadataKey = FingerprintMetadataKey
	}

	fingerprints := make(map[event.Type]string, len(declared))
	for _, declaration := range declared {
		fingerprints[declaration.eventType] = declaration.Fingerprint()
	}

	return func(events []event.Event) ([]event.Event, error) {
		stamped := make([]event.Event, len(events))

		for i, evt := range events {
			fingerprint, isDeclared := fingerprints[evt.Type()]
			if !isDeclared {
				stamped[i] = evt

				continue
			}

			if evt.Metadata().Custom[event.MetadataKey(metadataKey)] == fingerprint {
				stamped[i] = evt

				continue
			}

			upcasted, err := rebuildWithMetadataStamp(evt, metadataKey, fingerprint)
			if err != nil {
				return nil, err
			}

			stamped[i] = upcasted
		}

		return stamped, nil
	}
}

// rebuildWithMetadataStamp reconstructs the event with one added metadata
// entry, preserving identity (ID, stream, version, timestamp, encoding,
// schema version) and the payload bytes verbatim.
func rebuildWithMetadataStamp(
	evt event.Event,
	metadataKey, fingerprint string,
) (event.Event, error) {
	metadata := evt.Metadata().Clone().WithCustom(event.MetadataKey(metadataKey), fingerprint)

	return event.New(
		evt.Type(),
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		evt.Payload(),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithMetadata(metadata),
		event.WithEncoding(evt.Encoding()),
		event.WithSchemaVersion(evt.SchemaVersion()),
	)
}
