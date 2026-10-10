package schema

import (
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsid "github.com/larsartmann/go-cqrs-lite/id/v4"
	"pgregory.net/rapid"
)

// TestPropertyChain_VersionMonotonicityAndIdentity drives randomly generated
// 1:1 op sets (field ops and Transform — no Split/Drop, so cardinality stays
// 1:1) through a compiled chain and checks the invariants every read path
// relies on: version monotonicity (an op only ever advances the schema
// version) and identity preservation (the rebuild keeps event ID, timestamp,
// and stream identity so projection checkpoints stay aligned). Invalid random
// declarations are legitimate Compile rejections and skip the invariant.
func TestPropertyChain_VersionMonotonicityAndIdentity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		eventType := event.Type(rapid.SampledFrom(
			[]string{"user.created", "user.profiled", "balance.updated"},
		).Draw(rt, "eventType"))

		version := event.SchemaVersion(rapid.IntRange(1, 3).Draw(rt, "version"))

		// Each drawn op targets its OWN version (ascending), so Compile never
		// sees duplicates and the chain applies op after op — monotonicity is
		// then a multi-hop property, not a single-hop one.
		kinds := rapid.SliceOfN(rapid.IntRange(0, 3), 0, 3).Draw(rt, "kinds")

		ops := make([]Op, 0, len(kinds))

		for i, kind := range kinds {
			opVersion := version + event.SchemaVersion(i)

			switch kind {
			case 0:
				ops = append(ops, AddField(eventType, opVersion, "added", "default"))
			case 1:
				ops = append(ops, RemoveField(eventType, opVersion, "rank"))
			case 2:
				ops = append(ops, RenameField(eventType, opVersion, "name", "displayName"))
			case 3:
				ops = append(ops, Transform(eventType, opVersion, func(fields map[string]any) (map[string]any, error) {
					fields["derived"] = len(fields)

					return fields, nil
				}))
			}
		}

		chain, err := Compile(ops...)
		if err != nil {
			rt.Skip()
		}

		encoded, err := codec.JSONCodec{}.Encode(map[string]any{"name": "Lars", "rank": float64(3)})
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		streamID, err := cqrsid.ParseStreamID("01HK1540X0841Y0A6BSX1VKR95")
		if err != nil {
			t.Fatalf("parse stream id: %v", err)
		}

		source, err := event.NewEvent(
			eventType, streamID, "User", 1, encoded,
			event.WithSchemaVersion(version),
		)
		if err != nil {
			t.Fatalf("NewEvent: %v", err)
		}

		got, err := chain.upcastAll([]event.Event{source})
		if err != nil {
			t.Fatalf("upcastAll: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("1:1 ops must keep cardinality, got %d", len(got))
		}

		upcasted := got[0]

		if upcasted.SchemaVersion() < version {
			t.Fatalf("version regressed: %d < %d", upcasted.SchemaVersion(), version)
		}

		if upcasted.ID() != source.ID() {
			t.Fatal("event ID changed")
		}

		if !upcasted.OccurredAt().Equal(source.OccurredAt()) {
			t.Fatal("occurredAt changed")
		}

		if upcasted.StreamID() != source.StreamID() || upcasted.StreamType() != source.StreamType() {
			t.Fatal("stream identity changed")
		}
	})
}
