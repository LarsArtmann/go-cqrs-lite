package compat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func TestNewBaselineRendersCatalog(t *testing.T) {
	t.Parallel()

	baseline := NewBaseline(catalogWithEvents(
		eventMsg("order_placed", "1.0.0", oneField),
		eventMsg("order_shipped", "2.0.0", nil),
	))

	if baseline.FormatVersion != BaselineFormatVersion {
		t.Fatalf("formatVersion = %d, want %d", baseline.FormatVersion, BaselineFormatVersion)
	}

	if baseline.Catalog != "test" || baseline.CatalogVersion != "1.0.0" {
		t.Fatalf("catalog identity not captured: %+v", baseline)
	}

	if ids := baseline.MessageIDs(); len(ids) != 2 || ids[0] != "order_placed" ||
		ids[1] != "order_shipped" {
		t.Fatalf("MessageIDs() = %v, want sorted ids", ids)
	}

	if got := baseline.Messages["order_shipped"].Version; got != "2.0.0" {
		t.Fatalf("order_shipped version = %q", got)
	}

	if baseline.Messages["order_shipped"].Schema != nil {
		t.Fatal("a message without a schema must serialize without one")
	}

	if baseline.Messages["order_placed"].Kind != "event" {
		t.Fatalf("kind = %q, want event", baseline.Messages["order_placed"].Kind)
	}
}

func TestBaselineMarshalIsDeterministic(t *testing.T) {
	t.Parallel()

	cat := catalogWithEvents(
		eventMsg("b_event", "1.0.0", oneField),
		eventMsg("a_event", "1.0.0", twoFields),
	)

	first, err := NewBaseline(cat).MarshalIndent()
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	second, err := NewBaseline(cat).MarshalIndent()
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("baseline rendering is not deterministic:\n%s\n---\n%s", first, second)
	}

	if !strings.HasSuffix(string(first), "\n") {
		t.Fatal("baseline file must end with a newline")
	}
}

func TestBaselineRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "compat-baseline.json")
	cat := catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields))

	if err := NewBaseline(cat).WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	parsed, err := ParseBaseline(path)
	if err != nil {
		t.Fatalf("ParseBaseline: %v", err)
	}

	if report := CompareBaseline(parsed, cat); !report.OK() {
		t.Fatalf("a freshly written baseline must compare clean, got %s", report)
	}
}

func TestParseBaselineRejectsUnknownFormat(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "compat-baseline.json")
	write(t, path, `{"formatVersion": 99, "messages": {}}`)

	_, err := ParseBaseline(path)
	if err == nil {
		t.Fatal("want an error for an unknown formatVersion")
	}

	if !strings.Contains(err.Error(), "--update-baseline") {
		t.Fatalf("error must say how to fix it, got %q", err)
	}
}

func TestParseBaselineMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "absent.json")

	_, err := ParseBaseline(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("error must name the missing file, got %v", err)
	}
}

func TestParseBaselineRoundTripsExplicitMessages(t *testing.T) {
	t.Parallel()

	// Construct the row explicitly rather than through NewBaseline: this pins
	// the on-disk contract (field names, the schema pointer, the key/id
	// agreement) rather than whatever the renderer happens to emit today.
	original := &Baseline{
		FormatVersion:  BaselineFormatVersion,
		Catalog:        "orders",
		CatalogVersion: "3.1.0",
		Messages: map[string]BaselineMessage{
			"order_placed": {
				Kind:    "event",
				ID:      "order_placed",
				Version: "1.2.0",
				Schema:  rootObj(map[string]schema.Property{"id": strProp()}, "id"),
			},
			"order_cancelled": {
				Kind:    "event",
				ID:      "order_cancelled",
				Version: "1.0.0",
			},
		},
	}

	path := filepath.Join(t.TempDir(), "compat-baseline.json")
	if err := original.WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	parsed, err := ParseBaseline(path)
	if err != nil {
		t.Fatalf("ParseBaseline: %v", err)
	}

	if parsed.FormatVersion != BaselineFormatVersion {
		t.Fatalf("FormatVersion = %d, want %d", parsed.FormatVersion, BaselineFormatVersion)
	}

	withSchema := parsed.Messages["order_placed"]
	if withSchema.Version != "1.2.0" {
		t.Fatalf("order_placed version = %q, want 1.2.0", withSchema.Version)
	}

	if withSchema.Schema == nil {
		t.Fatal("order_placed lost its schema through the round trip")
	}

	if !propertiesEqual(withSchema.Schema.Properties["id"], strProp()) {
		t.Fatalf("order_placed schema = %+v, want the original property", withSchema.Schema)
	}

	if parsed.Messages["order_cancelled"].Schema != nil {
		t.Fatalf(
			"order_cancelled schema = %+v, want nil",
			parsed.Messages["order_cancelled"].Schema,
		)
	}
}

func TestCompareBaselineDetectsBreakingDrift(t *testing.T) {
	t.Parallel()

	baseline := NewBaseline(catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields)))
	current := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))

	report := CompareBaseline(baseline, current)

	blocking := report.Blocking()
	if len(blocking) != 1 || blocking[0].Kind != KindVersionNotBumped {
		t.Fatalf("want one version-not-bumped blocker, got %v (%s)", blocking, report)
	}
}

func TestCompareBaselineAdditiveAndLifecycle(t *testing.T) {
	t.Parallel()

	baseline := NewBaseline(catalogWithEvents(
		eventMsg("order_placed", "1.0.0", oneField),
		eventMsg("order_cancelled", "1.0.0", oneField),
	))
	current := catalogWithEvents(
		eventMsg("order_placed", "1.0.0", twoFields),
		eventMsg("order_refunded", "1.0.0", oneField),
	)

	report := CompareBaseline(baseline, current)

	if len(report.Blocking()) != 0 {
		t.Fatalf("no blocking findings expected, got %v", report.Blocking())
	}

	if !hasKind(report, KindFieldAdded) {
		t.Fatalf("missing field-added finding: %s", report)
	}

	if !hasKind(report, KindMessageAdded) || !hasKind(report, KindMessageRemoved) {
		t.Fatalf("missing lifecycle findings: %s (%v)", report, uniqueKinds(report))
	}
}

func TestCompareBaselineNilCatalog(t *testing.T) {
	t.Parallel()

	baseline := NewBaseline(catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField)))

	report := CompareBaseline(baseline, nil)
	if !hasKind(report, KindMessageRemoved) {
		t.Fatalf("an empty catalog must report every baseline message as removed: %s", report)
	}
}

func TestNewBaselineNilCatalog(t *testing.T) {
	t.Parallel()

	baseline := NewBaseline(nil)
	if baseline == nil || len(baseline.Messages) != 0 {
		t.Fatalf("nil catalog must yield an empty baseline, got %+v", baseline)
	}

	if got := baseline.MessageIDs(); len(got) != 0 {
		t.Fatalf("MessageIDs() = %v, want empty", got)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
