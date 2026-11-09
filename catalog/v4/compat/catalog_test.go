package compat

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func eventMsg(id, version string, sch *schema.Schema) catalog.Message {
	return catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        catalog.MessageID(id),
		Name:      catalog.Name(id),
		Version:   catalog.Version(version),
		Direction: catalog.Sends,
		Schema:    sch,
	}
}

func catalogWithEvents(msgs ...catalog.Message) *catalog.Catalog {
	return &catalog.Catalog{
		Title:   "test",
		Version: "1.0.0",
		Services: []catalog.Service{{
			ID:      "orders",
			Name:    "Orders",
			Version: "1.0.0",
			Events:  msgs,
		}},
	}
}

var (
	twoFields = rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()})
	oneField  = rootObj(map[string]schema.Property{"a": strProp()})
)

func TestCompatCatalogBlocksBreakingChangeWithoutVersionBump(t *testing.T) {
	t.Parallel()

	// The message version stays 1.0.0 while a field disappears. Existing
	// consumers' events lose a field they carry, and nothing records why.
	old := catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields))
	current := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))

	report := CompatCatalog(old, current)

	blocking := report.Blocking()
	if len(blocking) != 1 {
		t.Fatalf("want exactly one blocking finding, got %v (%s)", blocking, report)
	}

	if blocking[0].Kind != KindVersionNotBumped {
		t.Fatalf("blocking kind = %s, want %s", blocking[0].Kind, KindVersionNotBumped)
	}

	if blocking[0].Path != "order_placed" {
		t.Fatalf("blocking path = %q, want the message ID", blocking[0].Path)
	}

	if report.OK() {
		t.Fatal("a blocking report must not be OK")
	}
}

func TestCompatCatalogAllowsBreakingChangeWithVersionBump(t *testing.T) {
	t.Parallel()

	// Same shape change, but the declaration moved to 2.0.0: the author
	// acknowledged it. Whether the upcast op exists is cqrs-lint's half.
	old := catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields))
	current := catalogWithEvents(eventMsg("order_placed", "2.0.0", oneField))

	report := CompatCatalog(old, current)

	if got := len(report.Blocking()); got != 0 {
		t.Fatalf("version-bumped breaking change must not block, got %v", report.Blocking())
	}

	if report.Class != ClassNeedsVersionBump {
		t.Fatalf("class = %s, want %s", report.Class, ClassNeedsVersionBump)
	}
}

func TestCompatCatalogAdditiveChangeIsClean(t *testing.T) {
	t.Parallel()

	old := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))
	current := catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields))

	report := CompatCatalog(old, current)

	if !report.OK() {
		t.Fatalf("an additive change must be OK, got %s", report)
	}

	if report.Class != ClassAdditive {
		t.Fatalf("class = %s, want %s", report.Class, ClassAdditive)
	}
}

func TestCompatCatalogMessageLifecycle(t *testing.T) {
	t.Parallel()

	old := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))
	current := catalogWithEvents(eventMsg("order_shipped", "1.0.0", oneField))

	report := CompatCatalog(old, current)

	removed, ok := findKind(report, KindMessageRemoved)
	if !ok || removed.Path != "order_placed" {
		t.Fatalf("missing message-removed finding for order_placed: %s", report)
	}

	if removed.Blocking {
		t.Fatal("message removal is a domain decision and must not block")
	}

	added, ok := findKind(report, KindMessageAdded)
	if !ok || added.Path != "order_shipped" {
		t.Fatalf("missing message-added finding for order_shipped: %s", report)
	}

	if added.Class != ClassAdditive {
		t.Fatalf("message-added class = %s, want additive", added.Class)
	}
}

func TestCompatCatalogNilInputs(t *testing.T) {
	t.Parallel()

	if report := CompatCatalog(nil, nil); !report.OK() {
		t.Fatalf("nil catalogs must be OK, got %s", report)
	}

	report := CompatCatalog(nil, catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField)))
	if report.Class != ClassAdditive {
		t.Fatalf("populating an empty baseline class = %s, want additive", report.Class)
	}
}
