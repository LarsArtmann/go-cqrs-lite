package metaengine

import (
	"context"
	"fmt"
	"maps"
	"slices"
)

// RebuildThreshold configures when a projection rebuild is automatic vs
// requires operator confirmation (ADR-0124 §11).
type RebuildThreshold struct {
	// MaxEventCount is the maximum number of events a projection may contain
	// to qualify for automatic rebuild. Projections with more events require
	// explicit operator confirmation. Default: 100000.
	MaxEventCount int64

	// MaxDataBytes is the maximum estimated data size for automatic rebuild.
	// Projections larger than this require explicit confirmation.
	// Default: 1GB (1 << 30).
	MaxDataBytes int64
}

// DefaultRebuildThreshold returns the default rebuild threshold configuration.
func DefaultRebuildThreshold() RebuildThreshold {
	return RebuildThreshold{
		MaxEventCount: 100_000,
		MaxDataBytes:  1 << 30, // 1GB
	}
}

// LayoutDiff describes a single projection's layout change.
type LayoutDiff struct {
	QueryName string
	From      LayoutOption
	To        LayoutOption
	Reason    string

	// EstimatedRebuildEvents is the approximate number of events that must be
	// replayed to rebuild this projection.
	EstimatedRebuildEvents int64

	// EstimatedRebuildBytes is the approximate data size of the projection.
	EstimatedRebuildBytes int64

	// AutoRebuild is true when the projection is small enough to rebuild
	// automatically (below threshold). False means operator confirmation needed.
	AutoRebuild bool
}

// ReplanLayout applies an optional new operator priority config, re-plans
// through the same single replan path as Store.Replan, and returns the
// per-query layout diffs between the previous and the new plan.
//
// Convergence (ADR-0124 §5): routing and layout scoring happen in ONE
// planning pass (planQuery records both); this method no longer keeps a
// separate scoring copy. With pc == nil it re-plans under the current
// priority config (e.g. to pick up live latency calibration); with pc != nil
// it is equivalent to SetPriority followed by Replan, audited as
// "priority-change".
//
// Layout changes do NOT execute rebuilds — diffs with AutoRebuild=false are
// executed by ConfirmRebuild; AutoRebuild=true diffs are handled by Backfill.
func (s *Store) ReplanLayout(ctx context.Context, pc *PriorityConfig) ([]LayoutDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metaengine.Store.ReplanLayout: %w", err)
	}

	s.mu.Lock()
	if pc != nil {
		s.priorityConfig = pc
	}

	previous := s.layoutSnapshotLocked()
	s.mu.Unlock()

	trigger := triggerManual
	if pc != nil {
		trigger = triggerPriority
	}

	if err := s.replanWithTrigger(ctx, trigger); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.layoutDiffsLocked(previous), nil
}

// layoutSnapshotLocked captures each query's current layout from the active
// plan. Queries without a recorded layout default to LayoutEmbed (plans from
// before the Layout field existed). The caller must hold s.mu.
func (s *Store) layoutSnapshotLocked() map[string]LayoutOption {
	snapshot := make(map[string]LayoutOption)

	if s.plan == nil {
		return snapshot
	}

	for _, q := range s.plan.Queries {
		if q.Layout != "" {
			snapshot[q.QueryName] = q.Layout
		} else {
			snapshot[q.QueryName] = LayoutEmbed
		}
	}

	return snapshot
}

// layoutDiffsLocked diffs the new plan's layouts against a pre-replan
// snapshot, with rebuild estimates for every changed projection. The caller
// must hold s.mu (at least RLock).
func (s *Store) layoutDiffsLocked(previous map[string]LayoutOption) []LayoutDiff {
	threshold := DefaultRebuildThreshold()

	var diffs []LayoutDiff

	for _, qa := range s.plan.Queries {
		from, ok := previous[qa.QueryName]
		if !ok {
			from = LayoutEmbed
		}

		if qa.Layout == from {
			continue
		}

		q, ok := s.queries[qa.QueryName]
		if !ok {
			continue
		}

		vol := q.QueryConfig().Volume
		if vol <= 0 {
			vol = 1000
		}

		estimatedBytes := vol * 256 // ~256B per entry

		diffs = append(diffs, LayoutDiff{
			QueryName: qa.QueryName,
			From:      from,
			To:        qa.Layout,
			Reason: fmt.Sprintf(
				"priority change on %s engine (plan v%d)",
				qa.EngineName, s.plan.Version,
			),
			EstimatedRebuildEvents: vol,
			EstimatedRebuildBytes:  estimatedBytes,
			AutoRebuild: vol < threshold.MaxEventCount &&
				estimatedBytes < threshold.MaxDataBytes,
		})
	}

	return diffs
}

// ConfirmRebuild executes the layout rebuilds that require operator confirmation
// (diffs with AutoRebuild=false). It replays events from the attached EventLog
// into the affected projections, respecting the same idempotency safety as
// Backfill.
//
// Without an attached EventLog, ConfirmRebuild returns an error — it cannot
// rebuild projections without the event history.
//
// Auto-rebuild diffs (AutoRebuild=true) are NOT executed by this method; they
// are handled by ReplanLayout + Backfill when the threshold permits.
func (s *Store) ConfirmRebuild(ctx context.Context, diffs []LayoutDiff) error {
	confirmed := make([]LayoutDiff, 0, len(diffs))
	for _, d := range diffs {
		if !d.AutoRebuild {
			confirmed = append(confirmed, d)
		}
	}

	if len(confirmed) == 0 {
		return nil
	}

	if s.eventLog == nil {
		return fmt.Errorf(
			"metaengine.Store.ConfirmRebuild: %d diff(s) require rebuild but no EventLog is attached",
			len(confirmed),
		)
	}

	events := s.eventLog.Events()
	if len(events) == 0 {
		return nil
	}

	queryFilter := make(map[string]bool, len(confirmed))
	for _, d := range confirmed {
		queryFilter[d.QueryName] = true
	}

	return s.replayEvents(ctx, events, queryFilter, false)
}

// sortedQueryNames returns query names in sorted order for deterministic output.
func sortedQueryNames(queries map[string]queryMeta) []string {
	return slices.Sorted(maps.Keys(queries))
}
