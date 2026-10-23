package analyzer

import "testing"

// The wrapper-indirection false-positive class for CATALOG declarations
// (issue #79). Consumers declare their vocabulary through one generic helper
// that forwards a parameter as the message id, so the id constant only exists
// at the helper's call sites:
//
//	func registerEvent[P any](b *simple.Builder, agg string, t event.Type, s string) {
//		simple.Event[P](b, string(t), catalog.Sends, catalog.WithSummary(s))
//	}
//	registerEvent[contact.CreatedPayload](b, "contact", contact.EventCreated, "…")
//
// Reading only the helper body made the catalog look empty, so E004 flagged
// every emitted event on a catalog the consumer's own completeness test pins
// as exhaustive (CRM: 21 false positives).

func TestCatalogHelperIndirectionResolvesSimpleBuilderIDs(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"catalog.go": `package docs

import (
	catalog "github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const EventCreated event.Type = "contact.created"
const EventDeleted event.Type = "contact.deleted"

func registerEvent[P any](b *simple.Builder, agg string, t event.Type, summary string) {
	simple.Event[P](b, string(t), catalog.Sends,
		catalog.WithSummary(summary),
		catalog.Labels(map[string]string{"aggregate": agg}),
	)
}

func build() *catalog.Catalog {
	b := simple.New("svc", "1.0.0")
	registerEvent[struct{}](b, "contact", EventCreated, "created")
	registerEvent[struct{}](b, "contact", EventDeleted, "deleted")
	registerEvent[struct{}](b, "contact", "contact.merged", "merged")

	cat, _ := b.BuildE()

	return cat
}
`,
	})

	for _, want := range []string{"contact.created", "contact.deleted", "contact.merged"} {
		if !ctx.Registry.EventTypesInCatalog[want] {
			t.Errorf("helper-indirect catalog registration missing %q; catalog=%v",
				want, ctx.Registry.EventTypesInCatalog)
		}
	}
}

// A wrapper that never forwards a parameter (a literal inside the helper) must
// still register directly and must not create a helper entry that swallows
// unrelated call sites.
func TestCatalogHelperIndirectionLiteralHelperIsNotAHelper(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"catalog.go": `package docs

import (
	catalog "github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
)

func build() {
	b := simple.New("svc", "1.0.0")
	simple.Event[struct{}](b, "contact.created", catalog.Sends)
}
`,
	})

	if !ctx.Registry.EventTypesInCatalog["contact.created"] {
		t.Errorf("literal simple.Event registration missing; catalog=%v",
			ctx.Registry.EventTypesInCatalog)
	}

	if len(ctx.Registry.catalogHelperParams) != 0 {
		t.Errorf("literal-only catalog call must not register a helper; got %v",
			ctx.Registry.catalogHelperParams)
	}
}
