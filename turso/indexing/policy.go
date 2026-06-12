package indexing

// Policy controls how the advisor and auto-indexer handle different
// table categories. Allows consumers to customize recommendations and
// automatic index creation per table.
type Policy struct {
	// ExcludedTables is a set of table names that the advisor will
	// skip during analysis.
	ExcludedTables map[string]bool

	// CriticalTables is a set of tables where any recommendation is
	// considered Critical priority. Useful for high-traffic tables
	// where even optional indexes help.
	CriticalTables map[string]bool

	// SkipAutoCreate disables automatic index creation for the named
	// tables even if the advisor recommends indexes for them.
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
	if p ==nil {
		return false
	}

	return p.SkipAutoCreate[table]
}

// Exclude adds a table to the exclusion list.
func (p *Policy) Exclude(tables ...string) {
	for _, t := range tables {
		p.ExcludedTables[t] = true
	}
}

// MarkCritical adds tables to the critical list.
func (p *Policy) MarkCritical(tables ...string) {
	for _, t := range tables {
		p.CriticalTables[t] = true
	}
}

// MarkSkipAutoCreate adds tables to the skip-auto-create list.
func (p *Policy) MarkSkipAutoCreate(tables ...string) {
	for _, t := range tables {
		p.SkipAutoCreate[t] = true
	}
}
