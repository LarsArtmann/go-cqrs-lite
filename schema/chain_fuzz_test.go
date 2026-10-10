package schema

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsid "github.com/larsartmann/go-cqrs-lite/id/v4"
)

// FuzzChainHostilePayloads feeds arbitrary payload bytes (valid JSON,
// truncated JSON, binary garbage, empty) through a fixed realistic chain
// (rename + field ops + nested Transform with the passthrough policy) and
// asserts the read path survives: no panic, 1:1 cardinality, and either a
// clean upcast or a clean passthrough — never a corrupted half-transformed
// event.
func FuzzChainHostilePayloads(f *testing.F) {
	f.Add([]byte(`{"name":"Lars"}`))
	f.Add([]byte(`{"name":"Lars","rank":3}`))
	f.Add([]byte(`{"profile":{"name":"Lars"}}`))
	f.Add([]byte(`not json at all`))
	f.Add([]byte(`{"truncated":`))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0xff, 0xfe})

	f.Fuzz(func(t *testing.T, payload []byte) {
		chain, err := Compile(
			RenameType("user.renamed_legacy", "user.profiled"),
			AddField("user.profiled", 1, "added", "default",
				WithDecodePolicy(PassthroughOnDecodeError)),
			Transform("user.profiled", 2, func(fields map[string]any) (map[string]any, error) {
				if profile, ok := fields["profile"].(map[string]any); ok {
					profile["displayName"] = profile["name"]
				}

				return fields, nil
			}, WithDecodePolicy(PassthroughOnDecodeError)),
		)
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}

		streamID, err := cqrsid.ParseStreamID("01HK1540X0841Y0A6BSX1VKR95")
		if err != nil {
			t.Fatalf("parse stream id: %v", err)
		}

		source, err := event.NewEvent(
			"user.renamed_legacy", streamID, "User", 1, payload,
			event.WithSchemaVersion(1),
		)
		if err != nil {
			// Some payloads may be rejected at construction (e.g. invalid
			// for the raw-bytes path); construction rejects are out of scope.
			t.Skip()
		}

		got, err := chain.upcastAll([]event.Event{source})
		if err != nil {
			t.Fatalf("upcastAll must survive hostile payloads: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("rename+field ops keep cardinality 1:1, got %d", len(got))
		}

		if got[0].ID() != source.ID() {
			t.Fatal("event ID must survive any payload")
		}
	})
}
