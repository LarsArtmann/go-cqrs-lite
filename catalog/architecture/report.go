package architecture

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// ReportEntry is one line of an export check: a resource and the rule that
// put it in or out of the document (ADR-0160).
type ReportEntry struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ExportReport explains what an export keeps, redacts, drops and hides.
// Build one with [Exporter.Inspect]; the `catalog-export -check` flag
// renders it as text and exits non-zero when a selector matched nothing.
type ExportReport struct {
	// Included lists every resource in the export and why it is in:
	// "included", "in domain X", "in service X", "referenced by kind id",
	// or "whole catalog" when no include set was given.
	Included []ReportEntry
	// Excluded lists the redacted resources with their rule: "exclude" or
	// "neverExport". They stay hidden in every references mode.
	Excluded []ReportEntry
	// NotFound lists selectors that matched no resource or an unknown kind.
	// A non-empty list is the `-check` failure condition: a redaction that
	// matches nothing is a silent leak risk.
	NotFound []ReportEntry
	// Dropped lists the structured references a kept section makes to
	// resources outside the kept set; under [ReferencesOmit] they render
	// nothing, under [ReferencesNameOnly] they degrade to names.
	Dropped []ReportEntry
	// Hidden lists the suppressed reference parts in canonical order.
	Hidden []string
}

// Failed reports whether the check found a selector that matched nothing.
func (r ExportReport) Failed() bool {
	return len(r.NotFound) > 0
}

// Inspect resolves the export policy against the catalog WITHOUT rendering
// anything. It is the machine-readable half of the `-check` flag and costs
// one catalog walk.
func (e *Exporter) Inspect(cat *catalog.Catalog) ExportReport {
	enriched := cat.DeriveProducersConsumers()
	plan := e.cfg.sel.plan(enriched, e.cfg.references)

	report := ExportReport{
		Included: make([]ReportEntry, 0, len(plan.reasons)),
		Excluded: make([]ReportEntry, 0, len(plan.banned)),
		NotFound: make([]ReportEntry, 0, len(plan.notFound)),
		Dropped:  make([]ReportEntry, 0),
		Hidden:   make([]string, 0),
	}

	if plan.kept == nil {
		// No selection: the report still says what is in — everything.
		resolve := newSelectorResolver(enriched)

		for _, key := range resolve.allKeys() {
			if !plan.banned[key] {
				report.Included = append(report.Included, ReportEntry{
					Kind:   selectorSpelling(key.kind),
					ID:     key.id,
					Reason: "whole catalog",
				})
			}
		}
	}

	for key, reason := range plan.reasons {
		if plan.banned[key] {
			report.Excluded = append(report.Excluded, ReportEntry{
				Kind:   selectorSpelling(key.kind),
				ID:     key.id,
				Reason: reason,
			})
		} else if plan.kept[key] {
			report.Included = append(report.Included, ReportEntry{
				Kind:   selectorSpelling(key.kind),
				ID:     key.id,
				Reason: reason,
			})
		}
	}

	for _, sel := range plan.notFound {
		report.NotFound = append(report.NotFound, ReportEntry{
			Kind:   sel.Kind,
			ID:     sel.ID,
			Reason: "no such resource",
		})
	}

	if plan.kept != nil {
		resolve := newSelectorResolver(enriched)

		for key := range plan.kept {
			for _, edge := range resolve.edges(key) {
				ekind, ok := adrKind(edge.Kind)
				if !ok {
					continue
				}

				ekey := selKey{kind: ekind, id: edge.ID}

				if !plan.kept[ekey] {
					report.Dropped = append(report.Dropped, ReportEntry{
						Kind:   selectorSpelling(ekind),
						ID:     edge.ID,
						Reason: "omit",
					})
				}
			}
		}
	}

	for _, hidden := range []struct {
		on   bool
		name string
	}{
		{e.cfg.hide.owners, string(HideOwners)},
		{e.cfg.hide.repository, string(HideRepository)},
		{e.cfg.hide.examples, string(HideExamples)},
		{e.cfg.hide.specifications, string(HideSpecifications)},
		{e.cfg.hide.docs, string(HideDocs)},
	} {
		if hidden.on {
			report.Hidden = append(report.Hidden, hidden.name)
		}
	}

	sortEntries := func(entries []ReportEntry) {
		slices.SortFunc(entries, func(a, b ReportEntry) int {
			return cmp.Or(strings.Compare(a.Kind, b.Kind), strings.Compare(a.ID, b.ID))
		})
	}

	sortEntries(report.Included)
	sortEntries(report.Excluded)
	sortEntries(report.NotFound)
	sortEntries(report.Dropped)

	return report
}

// String renders the report as stable text: one header per non-empty
// section, one indented line per entry.
func (r ExportReport) String() string {
	var b strings.Builder

	line := func(format string, args ...any) {
		b.WriteString(fmt.Sprintf(format, args...))
		b.WriteByte('\n')
	}

	line("architecture reference check")

	section := func(title string, entries []ReportEntry) {
		if len(entries) == 0 {
			return
		}

		line("  %s: %d", title, len(entries))

		for _, entry := range entries {
			line("    %s %s — %s", entry.Kind, entry.ID, entry.Reason)
		}
	}

	section("included", r.Included)
	section("excluded", r.Excluded)
	section("dropped", r.Dropped)
	section("not found", r.NotFound)

	if len(r.Hidden) > 0 {
		line("  hidden: %s", strings.Join(r.Hidden, ", "))
	}

	if r.Failed() {
		line("  FAILED: %d selector(s) matched nothing", len(r.NotFound))
	}

	return b.String()
}
