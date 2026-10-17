package architecture_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// selectionExporter builds an exporter with the fixture's identity options
// plus the caller's selection options.
func selectionExporter(t *testing.T, opts ...architecture.Option) *architecture.Exporter {
	t.Helper()

	return architecture.NewExporter(append([]architecture.Option{
		architecture.WithTitle("Acme"),
		architecture.WithDescription("Acme's architecture."),
		architecture.WithGeneratedAt(excludeFrozenAt()),
		architecture.WithCommand("catalog-export -architecture"),
	}, opts...)...)
}

// renderSelection exports the fixture catalog under the given selection.
func renderSelection(t *testing.T, opts ...architecture.Option) string {
	t.Helper()

	out, err := selectionExporter(t, opts...).Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	return string(out)
}

// TestSelectionExcludeWins pins the exclude-first rule (ADR-0160): an exclude
// drops its resource even inside an included container, and excluding a
// container does not cascade to its members.
func TestSelectionExcludeWins(t *testing.T) {
	t.Run("exclude beats include inside an included domain", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "domain", ID: "ordering"}),
			architecture.WithExclude(architecture.Selector{Kind: "service", ID: "order-service"}),
		)

		for _, gone := range []string{
			`id="service-order-service"`,
			`id="command-create-order"`,
			`id="event-order-created"`,
			`id="query-get-order"`,
			`href="#service-order-service"`,
		} {
			if strings.Contains(out, gone) {
				t.Errorf("excluded service or its closure is present: %q", gone)
			}
		}

		if !strings.Contains(out, `id="domain-ordering"`) {
			t.Error("the included domain itself must stay")
		}
	})

	t.Run("exclude names exactly the listed resource, no cascade", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithExclude(architecture.Selector{Kind: "domain", ID: "ordering"}),
		)

		if strings.Contains(out, `id="domain-ordering"`) {
			t.Error("excluded domain is present")
		}

		for _, want := range []string{
			`id="service-order-service"`,
			`id="command-create-order"`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("exclude must not cascade to members, missing %q", want)
			}
		}
	})
}

// TestSelectionDomainBringsMembers pins the closure rule (ADR-0160): a domain
// selector brings its services and those services' messages; everything the
// closure does not name is absent.
func TestSelectionDomainBringsMembers(t *testing.T) {
	out := renderSelection(t,
		architecture.WithInclude(architecture.Selector{Kind: "domain", ID: "ordering"}),
	)

	for _, want := range []string{
		`id="domain-ordering"`,
		`id="service-order-service"`,
		`id="command-create-order"`,
		`id="event-order-created"`,
		`id="query-get-order"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("domain closure is missing %q", want)
		}
	}

	for _, gone := range []string{
		`id="channel-orders"`,
		`id="data-store-order-db"`,
		`id="flow-place-order"`,
		`id="entity-order"`,
		`id="adr-0001-events"`,
		`id="team-ordering"`,
		`id="doc-getting-started"`,
	} {
		if strings.Contains(out, gone) {
			t.Errorf("resource outside the closure is present: %q", gone)
		}
	}
}

// TestSelectionNeverExportBeatsInclude pins the strongest rule (ADR-0160):
// a never-exported resource is absent even when explicitly included or inside
// an included container.
func TestSelectionNeverExportBeatsInclude(t *testing.T) {
	t.Run("beats an explicit include", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "command", ID: "create-order"}),
			architecture.WithNeverExport(architecture.Selector{Kind: "command", ID: "create-order"}),
		)

		if strings.Contains(out, `id="command-create-order"`) {
			t.Error("never-exported command is present despite explicit include")
		}
	})

	t.Run("beats container closure, siblings survive", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "domain", ID: "ordering"}),
			architecture.WithNeverExport(architecture.Selector{Kind: "event", ID: "order-created"}),
		)

		if strings.Contains(out, `id="event-order-created"`) {
			t.Error("never-exported event is present via domain closure")
		}

		for _, want := range []string{
			`id="command-create-order"`,
			`id="query-get-order"`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("sibling message is missing: %q", want)
			}
		}
	})
}

// TestNoSelectionByteIdentical pins the compatibility claim: without any
// selection option (or with every rule list explicitly empty) the export is
// byte-identical to the whole-catalog reference of ADR-0159.
func TestNoSelectionByteIdentical(t *testing.T) {
	cat := fixtureCatalog()

	base, err := selectionExporter(t).Export(cat)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	for name, opts := range map[string][]architecture.Option{
		"no options":         nil,
		"empty include":      {architecture.WithInclude()},
		"empty exclude":      {architecture.WithExclude()},
		"empty never-export": {architecture.WithNeverExport()},
		"all empty": {
			architecture.WithInclude(),
			architecture.WithExclude(),
			architecture.WithNeverExport(),
		},
	} {
		got, err := selectionExporter(t, opts...).Export(cat)
		if err != nil {
			t.Fatalf("%s: Export: %v", name, err)
		}

		if !bytes.Equal(base, got) {
			t.Errorf("%s: export with an empty selection differs from the no-selection export", name)
		}
	}

	doc := string(base)

	for _, want := range []string{
		`id="domain-ordering"`,
		`id="service-order-service"`,
		`id="command-create-order"`,
		`id="event-order-created"`,
		`id="query-get-order"`,
		`id="channel-orders"`,
		`id="data-store-order-db"`,
		`id="flow-place-order"`,
		`id="entity-order"`,
		`id="adr-0001-events"`,
		`id="team-ordering"`,
		`id="doc-getting-started"`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("whole-catalog export is missing %q", want)
		}
	}
}
