package indexing

// Policy describes table categories for index advice. It is a standalone
// value: neither Advisor nor AutoIndexer reads one, because no option accepts
// it, so filling a Policy changes no recommendation and no automatic index
// creation. To exclude tables from analysis use WithExcludedTables instead.
type Policy struct {
	// ExcludedTables lists tables that a consumer of the Policy wants
	// skipped during analysis. Nothing reads it yet.
	ExcludedTables map[string]bool

	// CriticalTables lists tables whose recommendations a consumer of the
	// Policy wants treated as Critical priority. Nothing raises the
	// priority yet.
	CriticalTables map[string]bool

	// SkipAutoCreate lists tables for which a consumer of the Policy wants
	// no automatic index creation. Nothing skips them yet.
	SkipAutoCreate map[string]bool
}

// NewPolicy returns an empty Policy with all maps initialized.
func NewPolicy() *Policy {
	return &Policy{
		ExcludedTables: make(map[string]bool),
		CriticalTables: make(map[string]bool),
		SkipAutoCreate: make(map[string]bool),
	}
}

// ShouldExclude reports whether the policy excludes the given table.
func (p *Policy) ShouldExclude(table string) bool {
	if p == nil {
		return false
	}

	return p.ExcludedTables[table]
}

// IsCritical reports whether the policy treats the given table as critical.
func (p *Policy) IsCritical(table string) bool {
	if p == nil {
		return false
	}

	return p.CriticalTables[table]
}

// ShouldSkipAutoCreate reports whether the policy disables auto-creation
// for the given table.
func (p *Policy) ShouldSkipAutoCreate(table string) bool {
	if p == nil {
		return false
	}

	return p.SkipAutoCreate[table]
}

// Exclude adds a table to the exclusion list. A nil Policy or zero-value
// Policy (uninitialized maps) is handled gracefully: nil receivers are a
// no-op, and uninitialized maps are created on first write.
func (p *Policy) Exclude(tables ...string) {
	if p == nil {
		return
	}

	if p.ExcludedTables == nil {
		p.ExcludedTables = make(map[string]bool, len(tables))
	}

	for _, t := range tables {
		p.ExcludedTables[t] = true
	}
}

// MarkCritical adds tables to the critical list. Nil receivers are a no-op;
// uninitialized maps are created on first write.
func (p *Policy) MarkCritical(tables ...string) {
	if p == nil {
		return
	}

	if p.CriticalTables == nil {
		p.CriticalTables = make(map[string]bool, len(tables))
	}

	for _, t := range tables {
		p.CriticalTables[t] = true
	}
}

// MarkSkipAutoCreate adds tables to the skip-auto-create list. Nil
// receivers are a no-op; uninitialized maps are created on first write.
func (p *Policy) MarkSkipAutoCreate(tables ...string) {
	if p == nil {
		return
	}

	if p.SkipAutoCreate == nil {
		p.SkipAutoCreate = make(map[string]bool, len(tables))
	}

	for _, t := range tables {
		p.SkipAutoCreate[t] = true
	}
}
