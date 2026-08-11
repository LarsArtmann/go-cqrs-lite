package metaengine

import (
	"fmt"
	"strings"
)

// LayoutInfo describes the current physical layout of a query's projection
// (ADR-0124 Layer 4). This is surfaced by Store.GetLayoutInfo for observability.
type LayoutInfo struct {
	QueryName   string
	EngineName  string
	Layout      LayoutOption
	Priority    Priority
	ReadPattern ReadPattern
	Complexity  Complexity
}

// GetLayoutInfo returns the current layout information for all queries.
// This is the observability surface for layout planning (ADR-0124 §15).
func (s *Store) GetLayoutInfo() []LayoutInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]LayoutInfo, 0, len(s.queries))

	for _, name := range sortedQueryNames(s.queries) {
		q := s.queries[name]
		engine := q.QueryEngine()

		// Resolve the effective priority: developer per-query
		// WithLayoutPriority first, then operator PriorityConfig.
		resolvedPriority := PriorityBalanced
		if engine != nil {
			resolvedPriority = s.priorityForQuery(engine.Profile().Name, name, q.QueryConfig())
		}

		info := LayoutInfo{
			QueryName:   name,
			ReadPattern: q.QueryReadPattern(),
			Complexity:  q.QueryComplexity(),
			Priority:    resolvedPriority,
		}

		if engine != nil {
			profile := engine.Profile()
			info.EngineName = profile.Name
			info.Layout, _ = SelectLayout(profile, resolvedPriority)
		} else {
			info.Layout = LayoutEmbed
		}

		infos = append(infos, info)
	}

	return infos
}

// LayoutWarning represents a diagnostic warning about a layout decision
// (ADR-0124 §10). Warnings are advisory — the planner obeys but warns loudly.
type LayoutWarning struct {
	Type       string
	QueryName  string
	EngineName string
	Message    string
	Severity   string
}

const (
	// WarnPriorityMismatch: the resolved priority produces a suboptimal layout
	// for the engine (e.g., normalized on KV requires in-memory joins).
	WarnPriorityMismatch = "PRIORITY_MISMATCH"

	// WarnJoinAmplification: query requires multi-way joins on a non-SQL engine.
	WarnJoinAmplification = "JOIN_AMPLIFICATION"

	// WarnWriteAmplification: embedded layout with high child-mutation rate.
	WarnWriteAmplification = "WRITE_AMPLIFICATION"
)

// LayoutWarnings returns advisory warnings about current layout decisions.
// The planner obeys operator priorities but surfaces concerns through warnings.
//
// A warning is emitted ONLY when the resolved priority actually selects a
// normalized layout on a KV/LSM engine (which requires in-memory joins).
// When the priority selects Embed (the natural KV layout), no warning is
// emitted — the system is operating as designed.
func (s *Store) LayoutWarnings() []LayoutWarning {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var warnings []LayoutWarning

	for _, name := range sortedQueryNames(s.queries) {
		q := s.queries[name]
		engine := q.QueryEngine()
		if engine == nil {
			continue
		}

		profile := engine.Profile()
		storageLayout := defaultStorageLayout(profile)

		resolvedPriority := s.priorityForQuery(profile.Name, name, q.QueryConfig())

		selectedOption, _ := SelectLayout(profile, resolvedPriority)

		// Warn only when Normalize is selected on a KV/LSM engine.
		// Normalization on KV requires in-memory joins — expensive.
		if (storageLayout == LayoutKV || storageLayout == LayoutLSM) &&
			selectedOption == LayoutNormalize {
			warnings = append(warnings, LayoutWarning{
				Type:       WarnJoinAmplification,
				QueryName:  name,
				EngineName: profile.Name,
				Message: fmt.Sprintf(
					"priority=%s selected Normalize on %s engine — requires in-memory joins (consider ReadSpeed for this engine)",
					resolvedPriority,
					storageLayout,
				),
				Severity: "WARN",
			})
		}
	}

	return warnings
}

// layoutExplainAnnotation returns a compact layout+priority tag for ExplainPlan
// query lines, e.g. " layout=Embed(Balanced)". It is a pure function — callers
// must hold the store read lock (no re-locking to avoid deadlock).
func layoutExplainAnnotation(
	pc *PriorityConfig,
	profile EngineProfile,
	queryName string,
	cfg QueryConfig,
) string {
	resolved := PriorityBalanced
	if cfg.layoutPriority.Valid() {
		resolved = cfg.layoutPriority
	} else if pc != nil {
		resolved = pc.Resolve(profile.Name, queryName)
	}
	layout, _ := SelectLayout(profile, resolved)

	return fmt.Sprintf(" layout=%s(%s)", layout, resolved)
}

// LayoutDoctorSection returns the "--- Layout ---" text block for Doctor()
// output. Shows the resolved priority, layout option, and any warnings per query.
func (s *Store) LayoutDoctorSection() string {
	var b strings.Builder

	b.WriteString("\n--- Layout ---\n")

	infos := s.GetLayoutInfo()
	layoutAny := false

	for _, info := range infos {
		fmt.Fprintf(&b, "  %s: %s on %s (priority=%s, %s)\n",
			info.QueryName, info.Layout, info.EngineName,
			info.Priority, info.Complexity)
		layoutAny = true
	}

	if !layoutAny {
		b.WriteString("  no queries\n")
	}

	warnings := s.LayoutWarnings()
	if len(warnings) > 0 {
		b.WriteString("\n  Warnings:\n")

		for _, w := range warnings {
			fmt.Fprintf(&b, "    [%s] %s: %s\n", w.Severity, w.QueryName, w.Message)
		}
	}

	return b.String()
}
