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

// ReplanLayout computes new layouts for all queries given a priority config,
// identifies which projections must change, and returns the diffs.
//
// This is the operator's "what would happen if I changed the priority?" tool.
// It does NOT execute any rebuilds — it only computes the plan diff.
// Call ConfirmRebuild to execute the changes.
func (s *Store) ReplanLayout(ctx context.Context, pc *PriorityConfig) ([]LayoutDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metaengine.Store.ReplanLayout: %w", err)
	}

	threshold := DefaultRebuildThreshold()

	s.mu.RLock()
	defer s.mu.RUnlock()

	var diffs []LayoutDiff

	for _, name := range sortedQueryNames(s.queries) {
		q := s.queries[name]
		engine := q.QueryEngine()
		if engine == nil {
			continue
		}

		profile := engine.Profile()

		// Effective priority: developer WithLayoutPriority or operator config.
		// WithLayoutPriority set per-query is NOT overridden by ReplanLayout's
		// proposed pc (operator's what-if tool); the developer pin still wins
		// unless the operator explicitly overrides per-query.
		resolvedPriority := s.priorityForQuery(profile.Name, name, q.QueryConfig())
		if pc != nil {
			if p, ok := pc.PerQuery[name]; ok && p.Valid() {
				resolvedPriority = p
			} else if !q.QueryConfig().layoutPriority.Valid() {
				resolvedPriority = pc.Resolve(profile.Name, name)
			}
		}

		newOption, _ := SelectLayout(profile, resolvedPriority)

		// The current layout is Embed by default (existing behavior)
		currentOption := LayoutEmbed

		if newOption != currentOption {
			vol := q.QueryConfig().Volume
			if vol <= 0 {
				vol = 1000
			}

			estimatedBytes := vol * 256 // ~256B per entry

			diff := LayoutDiff{
				QueryName: name,
				From:      currentOption,
				To:        newOption,
				Reason: fmt.Sprintf(
					"priority=%s on %s engine",
					resolvedPriority,
					profile.Name,
				),
				EstimatedRebuildEvents: vol,
				EstimatedRebuildBytes:  estimatedBytes,
				AutoRebuild: vol < threshold.MaxEventCount &&
					estimatedBytes < threshold.MaxDataBytes,
			}

			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
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
