package systemscenario

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// journalTrail is the observable shape of a journal: per stream, the ordered
// (type, version, payload) triples. Minted event IDs and timestamps are
// ignored — two systems fed the same history are observationally equivalent
// exactly when these trails match (the system-level analog of
// scenario/v4's AssertObservationalEquivalence, ADR-0136 context).
type journalTrail map[string][]trailEntry

type trailEntry struct {
	eventType event.Type
	version   event.Version
	payload   []byte
}

// AssertJournalEquivalence fails when two booted systems' journals are not
// observationally equal — the deployment-swap proof: feed both systems the
// identical scenario (same DomainConfig, different DeploymentConfig
// engines) and the observable history must not diverge. A divergence is
// never flaky; it is one deployment's persistence distorting the record.
func AssertJournalEquivalence(t *testing.T, ctx context.Context, a, b *system.System) {
	t.Helper()

	trailA, err := readJournalTrail(ctx, a)
	if err != nil {
		t.Fatalf("journal equivalence: read system A: %v", err)
	}

	trailB, err := readJournalTrail(ctx, b)
	if err != nil {
		t.Fatalf("journal equivalence: read system B: %v", err)
	}

	if fmt.Sprint(trailA) != fmt.Sprint(trailB) {
		t.Fatalf("journal equivalence violated:\nsystem A: %s\nsystem B: %s",
			renderTrail(trailA), renderTrail(trailB))
	}
}

// readJournalTrail normalizes a system's journal into its observable trail.
func readJournalTrail(ctx context.Context, sys *system.System) (journalTrail, error) {
	journal, ok := sys.EventStore().(event.Journal)
	if !ok {
		return nil, fmt.Errorf("event store of %T does not implement event.Journal", sys.EventStore())
	}

	events, err := journal.ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("read journal: %w", err)
	}

	trail := make(journalTrail)
	for _, evt := range events {
		key := fmt.Sprintf("%s:%s", evt.StreamType(), evt.StreamID())
		trail[key] = append(trail[key], trailEntry{
			eventType: evt.Type(),
			version:   evt.Version(),
			payload:   evt.Payload(),
		})
	}

	return trail, nil
}

// renderTrail renders a trail for failure diagnostics.
func renderTrail(trail journalTrail) string {
	out := ""
	for stream, entries := range trail {
		out += "\n  " + stream + ":"

		for _, entry := range entries {
			out += fmt.Sprintf("\n    - %s v%d", entry.eventType, entry.version)
		}
	}

	return out
}
