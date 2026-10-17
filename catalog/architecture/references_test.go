package architecture_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// TestReferencesOmitDropsOutOfScope pins the default mode: with a selection,
// references to resources outside the kept set render nothing at all — no
// link and no name.
func TestReferencesOmitDropsOutOfScope(t *testing.T) {
	out := renderSelection(t,
		architecture.WithInclude(architecture.Selector{Kind: "service", ID: "order-service"}),
	)

	if !strings.Contains(out, `id="service-order-service"`) {
		t.Fatal("the included service itself is missing")
	}

	for _, gone := range []string{
		`href="#data-store-order-db"`,
		`href="#team-ordering"`,
		`href="#entity-order"`,
		`href="#channel-orders"`,
		"Order Database",
		"Ordering Team",
	} {
		if strings.Contains(out, gone) {
			t.Errorf("out-of-scope reference leaked under omit: %q", gone)
		}
	}
}

// TestReferencesNameOnlyKeepsNames pins name-only mode: out-of-scope
// references degrade to their display name as plain text, never to a link.
func TestReferencesNameOnlyKeepsNames(t *testing.T) {
	out := renderSelection(t,
		architecture.WithInclude(architecture.Selector{Kind: "service", ID: "order-service"}),
		architecture.WithReferences(architecture.ReferencesNameOnly),
	)

	for _, want := range []string{
		`<span class="rl rl-plain">Order Database</span>`,
		`<span class="rl rl-plain">Ordering Team</span>`,
		`<span class="rl rl-plain">Order</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("name-only mode is missing the plain-text name %q", want)
		}
	}

	for _, gone := range []string{
		`href="#data-store-order-db"`,
		`href="#team-ordering"`,
		`href="#entity-order"`,
	} {
		if strings.Contains(out, gone) {
			t.Errorf("name-only mode rendered a link outside the kept set: %q", gone)
		}
	}
}

// TestReferencesNameOnlyKeepsExcludedHidden pins the redaction rule that
// beats readability: a resource dropped by exclude or neverExport stays
// fully hidden in name-only mode too.
func TestReferencesNameOnlyKeepsExcludedHidden(t *testing.T) {
	out := renderSelection(t,
		architecture.WithInclude(architecture.Selector{Kind: "service", ID: "order-service"}),
		architecture.WithExclude(architecture.Selector{Kind: "container", ID: "order-db"}),
		architecture.WithReferences(architecture.ReferencesNameOnly),
	)

	if strings.Contains(out, "Order Database") {
		t.Error("an excluded resource's name leaked in name-only mode")
	}

	if !strings.Contains(out, `<span class="rl rl-plain">Ordering Team</span>`) {
		t.Error("an out-of-scope owner name should still render in name-only mode")
	}
}

// TestReferencesIncludePullsOneLevel pins the include mode of the references
// option: kept resources bring the resources they point at (one level), and
// a ban still wins over the pull.
func TestReferencesIncludePullsOneLevel(t *testing.T) {
	t.Run("service brings stores and entities", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "order-service"}),
			architecture.WithReferences(architecture.ReferencesInclude),
		)

		for _, want := range []string{
			`id="data-store-order-db"`,
			`id="entity-order"`,
			`id="service-order-service"`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("include mode is missing %q", want)
			}
		}

		if strings.Contains(out, `id="channel-orders"`) {
			t.Error("include mode pulled more than one level")
		}
	})

	t.Run("channel brings its messages", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "channel", ID: "orders"}),
			architecture.WithReferences(architecture.ReferencesInclude),
		)

		if !strings.Contains(out, `id="event-order-created"`) {
			t.Error("include mode did not pull the channel's message")
		}

		if strings.Contains(out, `id="command-create-order"`) {
			t.Error("include mode pulled an unrelated message")
		}
	})

	t.Run("exclude wins over the pull", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "order-service"}),
			architecture.WithExclude(architecture.Selector{Kind: "container", ID: "order-db"}),
			architecture.WithReferences(architecture.ReferencesInclude),
		)

		if strings.Contains(out, `id="data-store-order-db"`) {
			t.Error("an excluded store was pulled in by include mode")
		}

		if !strings.Contains(out, `id="entity-order"`) {
			t.Error("unrelated edge target is missing")
		}
	})
}

// TestSelectorSpellingsForEveryKind pins that the selector spellings the
// ADR-0160 doc comment promises actually resolve — decision/person/doc were
// silently ignored before adrKind learned them.
func TestSelectorSpellingsForEveryKind(t *testing.T) {
	t.Run("decision addresses an ADR", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "decision", ID: "0001-events"}),
		)

		if !strings.Contains(out, `id="adr-0001-events"`) {
			t.Error(`the "decision" selector spelling did not keep the ADR`)
		}

		if strings.Contains(out, `id="service-order-service"`) {
			t.Error("an unrelated service leaked into a decision-only export")
		}
	})

	t.Run("doc addresses a documentation page", func(t *testing.T) {
		out := renderSelection(t,
			architecture.WithInclude(architecture.Selector{Kind: "doc", ID: "getting-started"}),
		)

		if !strings.Contains(out, `id="doc-getting-started"`) {
			t.Error(`the "doc" selector spelling did not keep the page`)
		}
	})
}
