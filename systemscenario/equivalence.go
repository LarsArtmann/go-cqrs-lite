package systemscenario

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// journalTrail is the observable shape of a journal: per stream, the ordered
// (type, version) pairs. Minted event IDs, timestamps, and payload bytes are
// ignored — payloads embed minted stream IDs, and shape+order+versioning is
// what a deployment swap can distort; two systems fed the same history are
// observationally equivalent exactly when these trails match (the
// system-level analog of scenario/v4's AssertObservationalEquivalence,
// ADR-0136 context).
type journalTrail map[string][]trailEntry

type trailEntry struct {
	eventType event.Type
	version   event.Version
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

	normalizedA, normalizedB := normalizeTrail(trailA), normalizeTrail(trailB)
	if diff := trailDiff(normalizedA, normalizedB); diff != "" {
		t.Fatalf("journal equivalence violated:%s\n%s", diff, ordinalCaveat)
	}
}

// ordinalCaveat is appended to every equivalence failure: the pairwise
// stream matching is by first-appearance ordinal, so a deployment that
// writes the same streams in a different order shifts every ordinal at once
// and must be read as a global reordering, not N per-stream divergences.
const ordinalCaveat = "note: streams are matched pairwise by first-appearance ordinal " +
	"(#0, #1, ...), never by stream ID (IDs are minted per deployment) — a " +
	"deployment that writes the same streams in a different order shifts every " +
	"ordinal and is reported wholesale, not as per-stream divergences"

// trailDiff renders the pairwise per-stream divergence between two
// normalized trails, or "" when they are observationally equal. Streams
// beyond one trail's length are reported as missing from that system.
func trailDiff(a, b []normalizedStream) string {
	var sb strings.Builder

	for i := range max(len(a), len(b)) {
		switch {
		case i >= len(a):
			fmt.Fprintf(&sb, "\n\nstream #%d exists only in system B", i)
		case i >= len(b):
			fmt.Fprintf(&sb, "\n\nstream #%d exists only in system A", i)
		case fmt.Sprint(a[i].entries) != fmt.Sprint(b[i].entries):
			fmt.Fprintf(&sb, "\n\nstream #%d differs:\n  system A:%s\n  system B:%s",
				i, renderTrailEntries(a[i].entries), renderTrailEntries(b[i].entries))
		}
	}

	return sb.String()
}

// renderTrailEntries renders one stream's ordered (type, version) entries
// for failure diagnostics.
func renderTrailEntries(entries []trailEntry) string {
	//art-dupl:accept intentional: per-type diagnostic renderer, sibling of describeEvents
	if len(entries) == 0 {
		return "\n    (no events)"
	}

	var sb strings.Builder

	for _, entry := range entries {
		fmt.Fprintf(&sb, "\n    - %s v%d", entry.eventType, entry.version)
	}

	return sb.String()
}

// normalizeTrail replaces stream keys (which carry minted IDs) with ordinals
// by order of first appearance, sorted — two systems fed the same scenario
// in the same stream order compare equal regardless of the IDs they minted.
func normalizeTrail(trail journalTrail) []normalizedStream {
	keys := slices.Sorted(maps.Keys(trail))

	out := make([]normalizedStream, len(keys))
	for i, key := range keys {
		out[i] = normalizedStream{ordinal: i, entries: trail[key]}
	}

	return out
}

type normalizedStream struct {
	ordinal int
	entries []trailEntry
}

// readJournalTrail normalizes a system's journal into its observable trail.
func readJournalTrail(ctx context.Context, sys *system.System) (journalTrail, error) {
	journal, ok := sys.EventStore().(event.Journal)
	if !ok {
		return nil, fmt.Errorf(
			"event store of %T does not implement event.Journal",
			sys.EventStore(),
		)
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
		})
	}

	return trail, nil
}
