package schema

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"pgregory.net/rapid"
)

// propertyChain applies a generated 1:1 op set (field ops and Transform — no
// Split/Drop, so cardinality stays 1:1) through a compiled chain and checks
// the invariants every read path relies on.
func propertyChain(t *rapid.T, eventType event.Type, version event.SchemaVersion, ops []Op) {
	chain, err := Compile(ops...)
	if err != nil {
		// Invalid random declarations are legitimate Compile rejections;
		// only the valid ones carry the invariant.
		t.Skip()
	}

	payload := map[string]any{"name": "Lars", "rank": float64(3)}
	encoded, err := encodeFieldMapForTest(eventType, payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	source := newPayloadEvent(t, eventType, int(version), encoded)

	got, err := chain.upcastAll([]event.Event{source})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("1:1 ops must keep cardinality, got %d", len(got))
	}

	upcasted := got[0]

	// Version monotonicity: an op only ever advances the schema version.
	if upcasted.SchemaVersion() < version {
		t.Fatalf("version regressed: %d < %d", upcasted.SchemaVersion(), version)
	}

	// Identity preservation: the rebuild keeps the event's identity so
	// projection checkpoints and correlation stay aligned.
	if upcasted.ID() != source.ID() {
		t.Fatal("event ID changed")
	}

	if !upcasted.OccurredAt().Equal(source.OccurredAt()) {
		t.Fatal("occurredAt changed")
	}

	if upcasted.StreamID() != source.StreamID() || upcasted.StreamType() != source.StreamType() {
		t.Fatal("stream identity changed")
	}
}

func TestPropertyChainVersionMonotonicityAndIdentity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		eventType := event.Type(rapid.SampledFrom(
			"user.created", "user.profiled", "balance.updated",
		).Draw(t, "eventType"))

		version := event.SchemaVersion(rapid.IntRange(1, 3).Draw(t, "version"))

		var ops []Op

		if rapid.Bool().Draw(t, "addField") {
			ops = append(ops, AddField(eventType, version, "added", "default"))
		}

		if rapid.Bool().Draw(t, "removeField") {
			ops = append(ops, RemoveField(eventType, version, "rank"))
		}

		if rapid.Bool().Draw(t, "renameField") {
			ops = append(ops, RenameField(eventType, version, "name", "displayName"))
		}

		if rapid.Bool().Draw(t, "transform") {
			ops = append(ops, Transform(eventType, version, func(fields map[string]any) (map[string]any, error) {
				fields["derived"] = len(fields)

				return fields, nil
			}))
		}

		// No ops at all is a valid (identity) chain.
		propertyChain(t, eventType, version, ops)
	})
}
