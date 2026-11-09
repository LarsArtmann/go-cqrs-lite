package irohengine

import (
	"context"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// addOnlyGraphLocal implements graph dispatch (GraphAddEdge) but NOT the
// optional edge-removal extension — the local engine shape behind the
// record-but-skip contract in applyRemoteGraphRemove.
type addOnlyGraphLocal struct {
	added []metaengine.Edge
}

func (l *addOnlyGraphLocal) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "add-only"}
}

func (l *addOnlyGraphLocal) Close() error { return nil }

func (l *addOnlyGraphLocal) GraphAddEdge(
	_ context.Context,
	_ string,
	edge metaengine.Edge,
) error {
	l.added = append(l.added, edge)
	return nil
}

func (l *addOnlyGraphLocal) GraphNeighbors(
	_ context.Context,
	_ string,
	_ any,
	_ int,
) ([]any, error) {
	return nil, nil
}

// TestApplyRemoteGraphRemove_RecordsLWWWithoutBackend pins the
// record-but-skip contract: a remote edge-remove op aimed at a local engine
// without GraphRemoveEdge is skipped (nothing to remove) but its LWW
// timestamp is STILL recorded — so a stale reordered add carrying an older
// timestamp cannot resurrect the edge, and a newer add still applies.
func TestApplyRemoteGraphRemove_RecordsLWWWithoutBackend(t *testing.T) {
	t.Parallel()

	local := &addOnlyGraphLocal{}
	eng, ok := Replicated(local).(*replicatedEngine)
	if !ok {
		t.Fatalf("Replicated returned %T, want *replicatedEngine", eng)
	}

	base := time.Unix(5_000_000, 0)

	// Remote remove x→y at T1: the local engine cannot remove (no
	// GraphRemoveEdge) but must record the timestamp.
	eng.applyRemote(WriteOp{
		Collection: "follows",
		Kind:       OpGraphRemoveEdge,
		Timestamp:  base,
		Key:        "x",
		Value:      "y",
	})

	// Stale add x→y at T0 < T1 must be rejected by the LWW guard.
	eng.applyRemote(WriteOp{
		Collection: "follows",
		Kind:       OpGraphAddEdge,
		Timestamp:  base.Add(-time.Second),
		Key:        "x",
		Value:      "y",
	})
	if len(local.added) != 0 {
		t.Fatalf("stale add resurrected a removed edge: applied %v", local.added)
	}

	// A genuinely newer add (T2 > T1) must still apply — LWW recording
	// must not wedge the collection shut.
	eng.applyRemote(WriteOp{
		Collection: "follows",
		Kind:       OpGraphAddEdge,
		Timestamp:  base.Add(time.Second),
		Key:        "x",
		Value:      "z",
	})
	if len(local.added) != 1 || local.added[0].To != "z" {
		t.Fatalf("newer add must apply after recorded remove, got %v", local.added)
	}
}
