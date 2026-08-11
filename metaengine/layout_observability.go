package metaengine

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

		info := LayoutInfo{
			QueryName:   name,
			ReadPattern: q.QueryReadPattern(),
			Complexity:  q.QueryComplexity(),
			Layout:      LayoutEmbed, // current default
			Priority:    PriorityBalanced,
		}

		if engine != nil {
			info.EngineName = engine.Profile().Name
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

		// Check if the engine is KV/LSM and the layout would be normalized
		// (this requires in-memory joins — expensive on KV)
		layout := defaultStorageLayout(profile)

		if layout == LayoutKV || layout == LayoutLSM {
			// On KV engines, normalization requires in-memory joins
			// This is a potential warning if the operator chose WriteSpeed/StorageSpace
			// (which favor normalization on KV)
			warnings = append(warnings, LayoutWarning{
				Type:       WarnJoinAmplification,
				QueryName:  name,
				EngineName: profile.Name,
				Message:    "KV engine with normalized layout requires in-memory joins (consider ReadSpeed for this engine)",
				Severity:   "INFO",
			})
		}
	}

	return warnings
}
