package metaengine

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// PlanAuditTrigger values describe what caused a plan re-computation. They are
// recorded in PlanAuditEntry so operators can reconstruct "who changed what
// priority when" from the audit trail surfaced in Doctor() and PlanHistory().
const (
	triggerManual       = "manual"
	triggerPriority     = "priority-change"
	triggerEngineAdd    = "engine-added"
	triggerEngineRemove = "engine-removed"
	triggerAutoReroute  = "auto-reroute"
)

// maxPlanHistory bounds the in-memory audit ring so a long-running Store does
// not accumulate unbounded history.
const maxPlanHistory = 32

// PlanAuditEntry is one record in the plan audit trail. Each successful re-plan
// appends an entry capturing the resulting version, when it happened, what
// triggered it, and the active operator priority at that moment.
type PlanAuditEntry struct {
	// Version is the plan version produced by this re-plan (monotonic).
	Version int
	// At is when the re-plan completed (equals PlanResult.ComputedAt).
	At time.Time
	// Trigger names the cause: manual, priority-change, engine-added,
	// engine-removed, or auto-reroute.
	Trigger string
	// Priority is a snapshot of the active PriorityConfig (nil = no operator
	// priority set, resolves to Balanced everywhere).
	Priority *PriorityConfig
}

// PlanHistory returns the bounded audit trail of recent plan transitions, oldest
// first. At most maxPlanHistory entries are retained. Each entry records the
// plan version, timestamp, trigger, and the operator priority snapshot that was
// active when the plan was computed — answering "who changed what priority when"
// without requiring callers to intercept every Replan call.
func (s *Store) PlanHistory() []PlanAuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.planHistory)
}

// appendPlanAudit records one audit entry. Must be called with s.mu held for
// writing (called from the Phase-3 write-locked section of replanWithTrigger).
func (s *Store) appendPlanAudit(version int, at time.Time, trigger string, pc *PriorityConfig) {
	s.planHistory = append(s.planHistory, PlanAuditEntry{
		Version:  version,
		At:       at,
		Trigger:  trigger,
		Priority: clonePriorityConfig(pc),
	})

	if len(s.planHistory) > maxPlanHistory {
		s.planHistory = s.planHistory[len(s.planHistory)-maxPlanHistory:]
	}
}

// clonePriorityConfig returns a deep copy of pc (struct + map clones) so the
// audit trail is immune to later mutation of the live config. Returns nil for a
// nil input.
func clonePriorityConfig(pc *PriorityConfig) *PriorityConfig {
	if pc == nil {
		return nil
	}

	cp := *pc
	cp.PerEngine = clonePriorityMap(pc.PerEngine)
	cp.PerQuery = clonePriorityMap(pc.PerQuery)
	return &cp
}

func clonePriorityMap(m map[string]Priority) map[string]Priority {
	if m == nil {
		return nil
	}

	out := make(map[string]Priority, len(m))
	maps.Copy(out, m)
	return out
}

// formatPlanAuditTrail renders the most recent plan transitions as a compact
// one-line summary for the Doctor report. Returns "" when there is no history.
// Shows up to 5 recent entries, most-recent-first:
//
//	v3 priority-change(Balanced) ← v2 engine-added ← v1 manual
func (s *Store) formatPlanAuditTrail() string {
	s.mu.RLock()
	hist := s.planHistory
	s.mu.RUnlock()

	if len(hist) == 0 {
		return ""
	}

	const showLast = 5

	start := max(0, len(hist)-showLast)
	recent := hist[start:]
	parts := make([]string, 0, len(recent))

	for i := len(recent) - 1; i >= 0; i-- {
		e := recent[i]
		label := e.Trigger
		if e.Priority != nil {
			label += "(" + string(e.Priority.Resolve("", "")) + ")"
		}
		parts = append(parts, fmt.Sprintf("v%d %s", e.Version, label))
	}

	return strings.Join(parts, " ← ")
}
