package graph

import (
	"errors"
	"testing"
)

// An edge endpoint with a key prop the schema did not declare is the same
// typo as on MergeNode: reject it instead of silently creating a node under
// the wrong key.
func TestSchemaSink_RejectsEdgeEndpointWithWrongKeyProp(t *testing.T) {
	t.Parallel()

	endpoints := map[string]EdgeRef{
		"from": {
			Type: "AUTHORED_BY",
			From: NodeRef{Label: "Message", KeyProp: "slug", KeyValue: "m1"},
			To:   nodeOf("User", "u1"),
		},
		"to": {
			Type: "AUTHORED_BY",
			From: nodeOf("Message", "m1"),
			To:   NodeRef{Label: "User", KeyProp: "email", KeyValue: "u1"},
		},
	}

	for name, ref := range endpoints {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			driver := NewMemoryDriver(WithDriverSchema(testSchema()))

			err := driver.RunInTx(func(sink GraphSink) error {
				return sink.MergeEdge(ref, nil)
			})
			if !errors.Is(err, errSinkWrongKeyProp) {
				t.Fatalf("expected errSinkWrongKeyProp, got %v", err)
			}

			for _, node := range driver.Query(Pattern{}) {
				if node.Ref.KeyProp != "id" {
					t.Fatalf("rejected edge created a phantom node: %+v", node.Ref)
				}
			}
		})
	}
}

func nodeOf(label, key string) NodeRef {
	return NodeRef{Label: label, KeyProp: "id", KeyValue: key}
}
