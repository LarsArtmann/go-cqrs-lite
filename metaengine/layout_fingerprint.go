package metaengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// LayoutStampCollection is the reserved map collection where layout-plan
// fingerprints persist (one entry per projection collection, key = collection
// name, value = fingerprint). It rides the engine's ordinary map persistence,
// so it inherits ADR-0143 semantics: engine resets clear it with the other
// materialized collections (the journal survives), and an absent stamp is
// ACCEPTED — existing collections never fail a boot over missing history.
const LayoutStampCollection = "meta_layout_stamps"

// replayCompletePrefix marks, inside LayoutStampCollection, that a
// collection's replay finished (value = RFC3339 completion time). A stamp
// without a matching completed-replay marker means a rebuild may be
// mid-flight (crashed between reset and replay end) and the rebuild should
// run again before the collection is trusted.
const replayCompletePrefix = "__replay_complete__/"

// Fingerprint returns a stable identity of the layout plan: collection,
// table, sorted columns (name:type), and sorted indexes — SHA-256 over a
// deterministic serialization (an integrity fingerprint, never a cipher).
// Adding, removing, renaming, or re-typing any element changes it; the plan
// stays comparable across processes and runs.
func (p LayoutPlan) Fingerprint() string {
	columns := make([]string, 0, len(p.Columns))
	for _, column := range p.Columns {
		columns = append(columns, column.Name+":"+column.Type)
	}

	indexes := make([]string, 0, len(p.Indexes))
	for _, index := range p.Indexes {
		indexes = append(indexes, index.Name+"("+strings.Join(index.Columns, ",")+")")
	}

	sort.Strings(columns)
	sort.Strings(indexes)

	sum := sha256.Sum256([]byte(fmt.Sprintf(
		"v1|%s|%s|%s|%s",
		p.Collection, p.Table, strings.Join(columns, ";"), strings.Join(indexes, ";"),
	)))

	return "v1:" + hex.EncodeToString(sum[:8])
}

// LayoutStampDiff records one detected layout-plan drift: the fingerprint
// persisted for a collection disagrees with the currently declared plan.
// Recorded is the stored fingerprint; Declared is the fresh one.
type LayoutStampDiff struct {
	Collection string
	Recorded   string
	Declared   string
}

// RecordLayoutStamps persists the declared plans' fingerprints so a later
// boot can compare (see [LayoutStampDiffs]). Call after the layout plans are
// successfully applied — recording a plan that never landed would mask the
// next boot's drift signal.
func RecordLayoutStamps(
	ctx context.Context,
	backend MapBackend,
	plans []LayoutPlan,
) error {
	for _, plan := range plans {
		if err := backend.MapSet(
			ctx,
			LayoutStampCollection,
			plan.Collection,
			plan.Fingerprint(),
		); err != nil {
			return fmt.Errorf("metaengine.RecordLayoutStamps: %s: %w", plan.Collection, err)
		}
	}

	return nil
}

// LayoutStampDiffs compares the persisted fingerprints against the declared
// plans. Absent stamps produce NO diff (absent = accept: collections created
// before stamping, or cleared by an engine reset, boot clean); only a
// RECORDED fingerprint that disagrees with the declared plan drifts.
func LayoutStampDiffs(
	ctx context.Context,
	backend MapBackend,
	declared []LayoutPlan,
) ([]LayoutStampDiff, error) {
	var diffs []LayoutStampDiff

	for _, plan := range declared {
		recorded, found, err := backend.MapGet(ctx, LayoutStampCollection, plan.Collection)
		if err != nil {
			return nil, fmt.Errorf("metaengine.LayoutStampDiffs: %s: %w", plan.Collection, err)
		}

		if !found {
			continue
		}

		recordedFingerprint, _ := recorded.(string)
		if recordedFingerprint == "" || recordedFingerprint == plan.Fingerprint() {
			continue
		}

		diffs = append(diffs, LayoutStampDiff{
			Collection: plan.Collection,
			Recorded:   recordedFingerprint,
			Declared:   plan.Fingerprint(),
		})
	}

	return diffs, nil
}

// MarkReplayComplete records that the named collections' replay finished.
// Write-at-completion: call when a rebuild's replay returns cleanly — never
// before. A crash between reset and this marker leaves the rebuild visibly
// incomplete for the next boot.
func MarkReplayComplete(
	ctx context.Context,
	backend MapBackend,
	collections ...string,
) error {
	completedAt := time.Now().UTC().Format(time.RFC3339)

	for _, collection := range collections {
		if err := backend.MapSet(
			ctx,
			LayoutStampCollection,
			replayCompletePrefix+collection,
			completedAt,
		); err != nil {
			return fmt.Errorf("metaengine.MarkReplayComplete: %s: %w", collection, err)
		}
	}

	return nil
}

// ReplayCompletedAt reports when the collection's replay last completed
// (ok=false when no marker exists — treat an unmarked collection as
// potentially mid-rebuild, not as fresh).
func ReplayCompletedAt(
	ctx context.Context,
	backend MapBackend,
	collection string,
) (completedAt string, ok bool, err error) {
	value, found, err := backend.MapGet(ctx, LayoutStampCollection, replayCompletePrefix+collection)
	if err != nil || !found {
		return "", false, err
	}

	completedAt, _ = value.(string)

	return completedAt, completedAt != "", nil
}

// LayoutStampsDoctorSection renders the fingerprint + replay-marker state
// for every declared plan: the persisted fingerprint (or "absent"), whether
// it matches the declaration, and the last completed replay. One line per
// collection; empty string when nothing is declared.
func LayoutStampsDoctorSection(
	ctx context.Context,
	backend MapBackend,
	declared []LayoutPlan,
) string {
	if len(declared) == 0 {
		return ""
	}

	driftedByCollection := make(map[string]bool)
	diffs, err := LayoutStampDiffs(ctx, backend, declared)
	if err == nil {
		for _, diff := range diffs {
			driftedByCollection[diff.Collection] = true
		}
	}

	var section strings.Builder
	section.WriteString("--- Layout stamps ---\n")

	for _, plan := range declared {
		recorded, found, recordErr := backend.MapGet(ctx, LayoutStampCollection, plan.Collection)
		recordedFingerprint, _ := recorded.(string)

		state := "absent (accepted)"
		switch {
		case recordErr != nil:
			state = "error: " + recordErr.Error()
		case found && driftedByCollection[plan.Collection]:
			state = "DRIFT recorded=" + recordedFingerprint
		case found:
			state = "ok"
		}

		completedAt, completed, _ := ReplayCompletedAt(ctx, backend, plan.Collection)
		replayState := "no completed replay"
		if completed {
			replayState = "replayed at " + completedAt
		}

		section.WriteString(fmt.Sprintf(
			"%s: %s declared=%s %s\n",
			plan.Collection, state, plan.Fingerprint(), replayState,
		))
	}

	return section.String()
}
