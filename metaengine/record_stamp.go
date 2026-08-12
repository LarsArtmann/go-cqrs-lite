package metaengine

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// recordStamp pairs a destination field index (in the result struct) with a
// getter that extracts the value from a Record. Pre-computed at fold
// construction time.
type recordStamp struct {
	dstIdx int
	getter func(rec record.Record) any
}

// recordFieldGetters maps field names to getter functions that extract values
// from a Record. The type assertion (string/int64/int) matches the Go types
// the Record struct uses.
var recordFieldGetters = map[string]struct {
	getter func(rec record.Record) any
	typ    reflect.Type
}{
	"StreamID": {
		func(r record.Record) any { return string(r.StreamID) },
		reflect.TypeFor[string](),
	},
	"StreamType": {func(r record.Record) any { return r.StreamType }, reflect.TypeFor[string]()},
	"Version":    {func(r record.Record) any { return r.Version }, reflect.TypeFor[int64]()},
	"CorrelationID": {
		func(r record.Record) any { return r.MetaData.CorrelationID },
		reflect.TypeFor[string](),
	},
	"CausationID": {
		func(r record.Record) any { return r.MetaData.CausationID },
		reflect.TypeFor[string](),
	},
	"ActorID": {
		func(r record.Record) any { return r.MetaData.ActorID },
		reflect.TypeFor[string](),
	},
	"SchemaVersion": {
		func(r record.Record) any { return r.MetaData.SchemaVersion },
		reflect.TypeFor[int](),
	},
}

// computeRecordStamps finds result struct fields that match Record metadata
// fields by name + compatible type, excluding fields already covered by event
// mappings. This enables auto-folds to stamp Record metadata (StreamID,
// Version, CorrelationID, etc.) into results without explicit handler code.
func computeRecordStamps(
	resultType reflect.Type,
	eventMappings []fieldMapping,
) []recordStamp {
	covered := make(map[int]bool, len(eventMappings))
	for _, m := range eventMappings {
		covered[m.dstIdx] = true
	}

	var stamps []recordStamp

	for i := range resultType.NumField() {
		if covered[i] {
			continue
		}

		f := resultType.Field(i)
		if !f.IsExported() {
			continue
		}

		candidate, ok := recordFieldGetters[f.Name]
		if !ok {
			continue
		}

		if candidate.typ.AssignableTo(f.Type) {
			stamps = append(stamps, recordStamp{dstIdx: i, getter: candidate.getter})
		}
	}

	return stamps
}

// applyRecordStamps sets Record metadata fields on a result struct value.
// Record metadata is ALWAYS overwritten — it represents the current event's
// context (stream ID, version, correlation), not persistent entity state.
// Conflicts with event-mapped fields are prevented at construction time by
// computeRecordStamps, which excludes already-covered destination fields.
func applyRecordStamps(
	resultVal reflect.Value,
	stamps []recordStamp,
	rec record.Record,
) {
	for _, s := range stamps {
		resultVal.Field(s.dstIdx).Set(reflect.ValueOf(s.getter(rec)))
	}
}
