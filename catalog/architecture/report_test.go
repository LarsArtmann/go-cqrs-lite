package architecture_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// TestReportReasonsAndExitCode pins the -check contract: every entry carries
// its rule, a neverExport hit is reported as excluded, an edge target
// outside the kept set is reported as dropped, and a selector that matches
// nothing fails the check (the exit-1 condition).
func TestReportReasonsAndExitCode(t *testing.T) {
	exp := selectionExporter(t,
		architecture.WithInclude(
			architecture.Selector{Kind: "domain", ID: "ordering"},
		),
		architecture.WithExclude(
			architecture.Selector{Kind: "container", ID: "order-db"},
		),
		architecture.WithNeverExport(
			architecture.Selector{Kind: "event", ID: "order-created"},
		),
		architecture.WithHide(architecture.HideOwners),
	)

	report := exp.Inspect(fixtureCatalog())

	find := func(entries []architecture.ReportEntry, kind, id string) (architecture.ReportEntry, bool) {
		for _, entry := range entries {
			if entry.Kind == kind && entry.ID == id {
				return entry, true
			}
		}

		return architecture.ReportEntry{}, false
	}

	if entry, ok := find(report.Included, "domain", "ordering"); !ok || entry.Reason != "included" {
		t.Errorf("domain ordering: got %+v, ok=%v, want reason %q", entry, ok, "included")
	}

	if entry, ok := find(
		report.Included,
		"service",
		"order-service",
	); !ok ||
		entry.Reason != "in domain ordering" {
		t.Errorf("service via domain closure: got %+v, want %q", entry, "in domain ordering")
	}

	if entry, ok := find(
		report.Included,
		"command",
		"create-order",
	); !ok ||
		entry.Reason != "in service order-service" {
		t.Errorf("command via service closure: got %+v, want %q", entry, "in service order-service")
	}

	if entry, ok := find(
		report.Excluded,
		"container",
		"order-db",
	); !ok ||
		entry.Reason != "exclude" {
		t.Errorf("excluded store: got %+v, want %q", entry, "exclude")
	}

	if entry, ok := find(
		report.Excluded,
		"event",
		"order-created",
	); !ok ||
		entry.Reason != "neverExport" {
		t.Errorf("never-exported event: got %+v, want %q", entry, "neverExport")
	}

	if entry, ok := find(report.Dropped, "data-store", "order-db"); ok {
		t.Errorf("the excluded store must be excluded, not dropped: %+v", entry)
	}

	if len(report.Hidden) != 1 || report.Hidden[0] != "owners" {
		t.Errorf("hidden sections: got %v, want [owners]", report.Hidden)
	}

	if report.Failed() {
		t.Errorf("a fully matching selector set must not fail: %v", report.NotFound)
	}
}

// TestReportNotFoundFails pins the exit-1 condition: a selector that names
// no resource — or an unknown kind — fails the check.
func TestReportNotFoundFails(t *testing.T) {
	t.Run("missing resource", func(t *testing.T) {
		report := selectionExporter(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "ghost"}),
		).Inspect(fixtureCatalog())

		if !report.Failed() {
			t.Error("a selector matching nothing must fail the check")
		}

		if len(report.NotFound) != 1 || report.NotFound[0].ID != "ghost" {
			t.Errorf("not-found entries: %+v", report.NotFound)
		}

		if !strings.Contains(report.String(), "FAILED") {
			t.Error("the text report must surface the failure")
		}
	})

	t.Run("unknown kind", func(t *testing.T) {
		report := selectionExporter(t,
			architecture.WithNeverExport(architecture.Selector{Kind: "gadget", ID: "x"}),
		).Inspect(fixtureCatalog())

		if !report.Failed() {
			t.Error("an unknown kind must fail the check — a redaction that matches nothing leaks")
		}
	})

	t.Run("no selection never fails", func(t *testing.T) {
		report := selectionExporter(t).Inspect(fixtureCatalog())

		if report.Failed() {
			t.Error("the whole-catalog export has nothing to fail on")
		}

		if len(report.Included) == 0 {
			t.Error("the whole-catalog report must list included resources")
		}
	})
}

// TestReportDroppedListsEdgeTargets pins the dropped section: structured
// references from kept resources to out-of-set resources are listed with
// the omit reason.
func TestReportDroppedListsEdgeTargets(t *testing.T) {
	report := selectionExporter(t,
		architecture.WithInclude(architecture.Selector{Kind: "service", ID: "order-service"}),
	).Inspect(fixtureCatalog())

	found := false

	for _, entry := range report.Dropped {
		if entry.ID == "order-db" {
			if entry.Reason != "omit" {
				t.Errorf("dropped store reason: %q, want omit", entry.Reason)
			}

			found = true
		}
	}

	if !found {
		t.Error("the store the service reads from is missing from the dropped list")
	}
}
