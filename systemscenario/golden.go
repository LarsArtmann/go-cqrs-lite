package systemscenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Trail renders the deterministic one-line-per-event trail of a scenario:
// type, version, and actor when stamped. Minted IDs and timestamps are
// deliberately excluded — the trail of a fixed scenario is byte-stable
// across runs, which is what golden files pin.
func Trail(events []event.Event) string {
	if len(events) == 0 {
		return "(no events)\n"
	}

	lines := make([]string, len(events))
	for i, evt := range events {
		line := fmt.Sprintf("- %s v%d", evt.Type(), evt.Version())
		if actor := evt.Metadata().ActorID.String(); actor != "" {
			line += " actor=" + actor
		}

		lines[i] = line
	}

	return strings.Join(lines, "\n") + "\n"
}

// ThenGolden asserts the act events' [Trail] against a go-snaps golden
// snapshot at path (filename relative to the test's snapshot directory —
// the file is placed next to it under __snapshots__). Update all goldens
// with UPDATE_SNAPS=true go test ./...; clean obsolete ones with
// UPDATE_SNAPS=clean (go-snaps' standard workflow, same as eventtest).
//
//	sc.Given(...).When(cmd).ThenGolden(t, "task_lifecycle")
func (p *WhenPhase) ThenGolden(t *testing.T, path string) *WhenPhase {
	t.Helper()

	s := p.sc
	s.requireAct("ThenGolden")

	snaps.WithConfig(
		snaps.Dir("__snapshots__"),
		snaps.Filename(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
	).MatchSnapshot(t, Trail(s.actEvents()))

	return p
}

// TrailWithPayloads renders the [Trail] plus one payload hash per event:
// each payload's JSON encoding, with every masked ID replaced by a
// first-appearance ordinal token (<<stream#0>>, <<stream#1>>, ...), is
// sha256-hashed and rendered as a payload= suffix. Masking runs BEFORE the
// hash so payloads that embed minted stream IDs — fresh ULIDs every run —
// cannot destabilize a golden: the token depends only on WHERE an ID first
// appears in the trail, and the scenario fixes that order. Masks are matched
// longest-first so no ID shadows a substring of another.
//
// Pass the stream IDs the scenario minted ([Scenario] tracks them
// automatically for ThenGoldenWithPayloads); extra IDs a payload embeds
// without owning a stream can be added to masks.
func TrailWithPayloads(events []event.Event, masks []string) string {
	if len(events) == 0 {
		return "(no events)\n"
	}

	masked := nonEmptyMasks(masks)
	encoded := make([][]byte, len(events))

	// First appearance order across the whole trail — the ordinal source.
	tokens := map[string]string{}

	for i, evt := range events {
		// Payload() is the stored encoding ([]byte — CBOR by default);
		// its text strings appear verbatim, so masking operates on those
		// bytes directly.
		encoded[i] = evt.Payload()

		for _, m := range masked {
			_, seen := tokens[m]
			if !seen && bytes.Contains(encoded[i], []byte(m)) {
				tokens[m] = maskToken(len(tokens))
			}
		}
	}

	lines := make([]string, len(events))
	for i, evt := range events {
		line := fmt.Sprintf("- %s v%d", evt.Type(), evt.Version())
		if actor := evt.Metadata().ActorID.String(); actor != "" {
			line += " actor=" + actor
		}

		line += " payload=" + hashMasked(encoded[i], masked, tokens)
		lines[i] = line
	}

	return strings.Join(lines, "\n") + "\n"
}

// ThenGoldenWithPayloads asserts [TrailWithPayloads] against a go-snaps
// golden, masking the scenario's tracked stream IDs (plus any extraMasks)
// before hashing — use it instead of [WhenPhase.ThenGolden] when payloads
// embed minted IDs. Update with UPDATE_SNAPS=true go test ./... like
// ThenGolden.
//
//	sc.Given(...).When(cmd).ThenGoldenWithPayloads(t, "task_lifecycle")
func (p *WhenPhase) ThenGoldenWithPayloads(
	t *testing.T,
	path string,
	extraMasks ...string,
) *WhenPhase {
	t.Helper()

	s := p.sc
	s.requireAct("ThenGoldenWithPayloads")

	masks := append(s.trackedStreamMasks(), extraMasks...)

	snaps.WithConfig(
		snaps.Dir("__snapshots__"),
		snaps.Filename(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
	).MatchSnapshot(t, TrailWithPayloads(s.actEvents(), masks))

	return p
}

// maskToken renders the placeholder substituted for the Nth distinct masked
// ID. Distinctive delimiters keep a payload's literal text from colliding.
func maskToken(ordinal int) string {
	return fmt.Sprintf("<<stream#%d>>", ordinal)
}

// nonEmptyMasks dedupes and sorts masks longest-first so an ID that is a
// substring of another is replaced before it can be clobbered.
func nonEmptyMasks(masks []string) []string {
	out := make([]string, 0, len(masks))
	for _, m := range masks {
		if m != "" && !slices.Contains(out, m) {
			out = append(out, m)
		}
	}

	slices.SortFunc(out, func(a, b string) int { return len(b) - len(a) })

	return out
}

// hashMasked substitutes every mask with its ordinal token (longest-first)
// and returns the sha256 hex of the result.
func hashMasked(encoded []byte, masked []string, tokens map[string]string) string {
	for _, m := range masked {
		token, seen := tokens[m]
		if !seen {
			continue
		}

		encoded = bytes.ReplaceAll(encoded, []byte(m), []byte(token))
	}

	sum := sha256.Sum256(encoded)

	return hex.EncodeToString(sum[:])
}

// trackedStreamMasks collects the stream IDs the scenario has seen so far —
// versionHints keys are "Type:ID" stream refs.
func (s *Scenario) trackedStreamMasks() []string {
	masks := make([]string, 0, len(s.versionHints))

	for key := range s.versionHints {
		if _, id, ok := strings.Cut(key, ":"); ok {
			masks = append(masks, id)
		}
	}

	return masks
}
