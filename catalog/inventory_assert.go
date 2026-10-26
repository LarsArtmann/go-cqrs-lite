package catalog

import (
	"fmt"
	"slices"
	"strings"
)

// TestingT is the minimal test surface AssertInventory needs. testing.T,
// testing.B, and testing.F satisfy it structurally; declaring it here keeps
// the catalog module free of a "testing" import in production code.
type TestingT interface {
	Helper()
	Errorf(format string, args ...any)
}

// AssertInventory fails the test when the catalog's complete message
// inventory (commands + events + queries across ALL services, compared as a
// set of message IDs) differs from want. It is the drift gate for derived
// catalogs: pin the expected inventory in a test, derive the catalog from
// declared truth, and any added/removed message fails the build with a
// report of exactly what drifted.
//
//	want := []string{"invoice.record_state", "invoice.issued", "invoice.updated"}
//	catalog.AssertInventory(t, readmodels.BuildCatalog(), want...)
func AssertInventory(t TestingT, cat *Catalog, want ...string) {
	t.Helper()

	assertInventory(t, cat, want, collectAll, "messages")
}

// AssertCommandInventory asserts only the command inventory (same rules as
// [AssertInventory]).
func AssertCommandInventory(t TestingT, cat *Catalog, want ...string) {
	t.Helper()

	assertInventory(t, cat, want, collectCommands, "commands")
}

// AssertEventInventory asserts only the event inventory (same rules as
// [AssertInventory]).
func AssertEventInventory(t TestingT, cat *Catalog, want ...string) {
	t.Helper()

	assertInventory(t, cat, want, collectEvents, "events")
}

// AssertQueryInventory asserts only the query inventory (same rules as
// [AssertInventory]).
func AssertQueryInventory(t TestingT, cat *Catalog, want ...string) {
	t.Helper()

	assertInventory(t, cat, want, collectQueries, "queries")
}

func assertInventory(
	t TestingT,
	cat *Catalog,
	want []string,
	collect func(*Catalog) []string,
	kindLabel string,
) {
	t.Helper()

	if cat == nil {
		t.Errorf("AssertInventory: catalog is nil — derive or build it before asserting")

		return
	}

	actual := collect(cat)
	sortedWant := slices.Clone(want)
	slices.Sort(sortedWant)

	missing := difference(sortedWant, actual)
	unexpected := difference(actual, sortedWant)

	if len(missing) == 0 && len(unexpected) == 0 {
		return
	}

	var report strings.Builder

	fmt.Fprintf(&report, "catalog %s inventory drifted (want %d, got %d):", kindLabel, len(sortedWant), len(actual))

	if len(missing) > 0 {
		fmt.Fprintf(&report, "\n  missing (declared in want, absent from catalog): %s",
			strings.Join(missing, ", "))
	}

	if len(unexpected) > 0 {
		fmt.Fprintf(&report, "\n  unexpected (in catalog, not declared in want): %s",
			strings.Join(unexpected, ", "))
	}

	report.WriteString("\n  fix: update the want list if the change is intended, otherwise fix the derivation/hand-maintained list")

	t.Errorf("%s", report.String())
}

// collectAll returns the sorted, deduplicated inventory of every message ID
// across all services.
func collectAll(cat *Catalog) []string {
	seen := map[string]struct{}{}

	for _, id := range collectCommands(cat) {
		seen[id] = struct{}{}
	}

	for _, id := range collectEvents(cat) {
		seen[id] = struct{}{}
	}

	for _, id := range collectQueries(cat) {
		seen[id] = struct{}{}
	}

	return sortedKeys(seen)
}

func collectCommands(cat *Catalog) []string {
	seen := map[string]struct{}{}

	for _, svc := range cat.Services {
		for _, cmd := range svc.Commands {
			seen[string(Key(cmd))] = struct{}{}
		}
	}

	return sortedKeys(seen)
}

func collectEvents(cat *Catalog) []string {
	seen := map[string]struct{}{}

	for _, svc := range cat.Services {
		for _, evt := range svc.Events {
			seen[string(Key(evt))] = struct{}{}
		}
	}

	return sortedKeys(seen)
}

func collectQueries(cat *Catalog) []string {
	seen := map[string]struct{}{}

	for _, svc := range cat.Services {
		for _, qry := range svc.Queries {
			seen[string(Key(qry))] = struct{}{}
		}
	}

	return sortedKeys(seen)
}

func sortedKeys(seen map[string]struct{}) []string {
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}

	slices.Sort(out)

	return out
}

// difference returns the sorted elements of a absent from b (both must be
// sorted; duplicates collapse via the caller's set semantics).
func difference(a, b []string) []string {
	var out []string

	for _, item := range a {
		if !slices.Contains(b, item) {
			out = append(out, item)
		}
	}

	return out
}
