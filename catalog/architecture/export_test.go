package architecture_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

type createOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"`
	Amount  int    `json:"amount" doc:"Total in minor units"`
}

type orderCreated struct {
	OrderID string `json:"order_id" doc:"Order identifier"`
	Amount  int    `json:"amount" doc:"Total in minor units"`
}

type getOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"`
}

// fixtureCatalog builds a small but complete catalog: a domain, a service, one
// of each message kind, a channel, a data store, a flow, an entity, a team,
// a decision record and a documentation page.
func fixtureCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("Acme", "1.0.0")

	builder.AddService("order-service", "Order Service", "1.0.0", "Takes and tracks orders.",
		catalog.Command[createOrder]("create-order",
			catalog.WithSummary("Create an order."),
			catalog.MsgOperation("POST", "/orders"),
		),
		catalog.Event[orderCreated]("order-created", catalog.Sends,
			catalog.WithSummary("An order was created."),
		),
		catalog.Query[getOrder]("get-order",
			catalog.WithSummary("Fetch one order."),
		),
	)

	builder.ConfigureService("order-service",
		catalog.ServiceOwners("ordering"),
		catalog.ServiceWritesTo("order-db"),
		catalog.ServiceReadsFrom("order-db"),
		catalog.ServiceEntities("order"),
	)

	builder.AddDomain("ordering", "Ordering", "1.0.0", "Everything about orders.", "order-service")
	builder.ConfigureDomain("ordering",
		catalog.DomainUbiquitousLanguage(catalog.UbiquitousLanguageTerm{
			Name:        catalog.Name("Order"),
			Description: "A customer's request to buy.",
		}),
	)

	builder.AddChannel(catalog.Channel{
		ID:      "orders",
		Name:    "Orders",
		Version: "1.0.0",
		Summary: "Order lifecycle events.",
	})
	builder.ConfigureChannel("orders", catalog.ChannelMessages("order-created"))

	builder.AddDataStore(catalog.DataStore{
		ID:            "order-db",
		Name:          "Order Database",
		Version:       "1.0.0",
		Summary:       "System of record for orders.",
		Technology:    "postgres@16",
		Authoritative: true,
	})

	builder.AddFlow(catalog.Flow{
		ID:      "place-order",
		Name:    "Place an order",
		Version: "1.0.0",
		Steps: []catalog.FlowStep{
			{
				ID:      "create",
				Title:   "Create the order",
				Service: &catalog.Ref{ID: "order-service"},
				Message: &catalog.Ref{ID: "create-order"},
			},
			{
				ID:      "publish",
				Title:   "Publish the event",
				Message: &catalog.Ref{ID: "order-created"},
				NextStep: &catalog.FlowEdge{
					ID:    "publish",
					Label: "done",
				},
			},
		},
	})

	builder.AddEntity(catalog.Entity{
		ID:            "order",
		Name:          "Order",
		Version:       "1.0.0",
		Summary:       "A customer order.",
		AggregateRoot: true,
		Properties: []catalog.EntityProperty{
			{Name: "id", Type: "string", Required: true, Description: "Order identifier."},
		},
		Schema: catalog.SchemaFromType[orderCreated](),
	})

	builder.AddADR(catalog.ADR{
		ID:      "0001-events",
		Name:    "Model orders with events",
		Version: "1.0.0",
		Status:  catalog.ADRAccepted,
		Date:    "2026-01-02",
		Summary: "Events are the order history.",
		Body:    "# Decision\n\nUse **events** for the order lifecycle.\n\n- replayable\n- auditable\n",
		AppliesTo: []catalog.ADRResourceRef{
			{Type: catalog.ADRAppliesToService, ID: "order-service"},
		},
	})

	builder.AddTeam(catalog.Team{ID: "ordering", Name: "Ordering Team", Summary: "Owns ordering."})
	builder.AddCustomDoc(catalog.CustomDoc{
		ID:      "getting-started",
		Title:   "Getting started",
		Summary: "How to read this reference.",
		Content: "Start with the [Order Service](#service-order-service), then read the #event-order-created event.\n",
	})

	return builder.Build()
}

// excludeFrozenAt pins the generation time so golden-ish assertions stay stable.
func excludeFrozenAt() time.Time {
	return time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
}

// fixtureExporter builds the exporter every render test shares.
func fixtureExporter() *architecture.Exporter {
	return architecture.NewExporter(
		architecture.WithTitle("Acme"),
		architecture.WithDescription("Acme's architecture."),
		architecture.WithGeneratedAt(excludeFrozenAt()),
		architecture.WithCommand("catalog-export -architecture"),
	)
}

func renderFixture(t *testing.T) string {
	t.Helper()

	out, err := fixtureExporter().Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	return string(out)
}

func TestExportProducesSelfContainedDocument(t *testing.T) {
	out := renderFixture(t)

	for _, want := range []string{
		"<!doctype html>",
		"<style>",
		"<script>",
		`<section class="section" id="domain-ordering"`,
		`<section class="section" id="service-order-service"`,
		`<section class="section" id="command-create-order"`,
		`<section class="section" id="event-order-created"`,
		`<section class="section" id="query-get-order"`,
		`<section class="section" id="channel-orders"`,
		`<section class="section" id="data-store-order-db"`,
		`<section class="section" id="flow-place-order"`,
		`<section class="section" id="entity-order"`,
		`<section class="section" id="adr-0001-events"`,
		`<section class="section" id="team-ordering"`,
		`<section class="section" id="doc-getting-started"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q", want)
		}
	}
}

func TestExportLinksResourcesToEachOther(t *testing.T) {
	out := renderFixture(t)

	for _, want := range []string{
		`href="#service-order-service"`,
		`href="#event-order-created"`,
		`href="#data-store-order-db"`,
		`href="#channel-orders"`,
		`href="#team-ordering"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output has no link %q", want)
		}
	}
}

func TestExportRendersMessageSchema(t *testing.T) {
	out := renderFixture(t)

	for _, want := range []string{
		"order_id",
		"Order identifier",
		"export interface OrderCreated",
		"Example payload",
		"JSON Schema",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("message section is missing %q", want)
		}
	}
}

func TestExportEmbedsSearchIndex(t *testing.T) {
	out := renderFixture(t)

	if !strings.Contains(out, `id="search-index"`) {
		t.Fatal("search index script is missing")
	}

	if !strings.Contains(out, `"h":"event-order-created"`) {
		t.Error("search index does not carry the event anchor")
	}
}

func TestExportEscapesCatalogData(t *testing.T) {
	builder := catalog.NewBuilder("Hostile", "1.0.0")
	builder.AddService("evil", `<script>alert(1)</script>`, "1.0.0", `" onload="alert(2)`)
	builder.AddService("evil2", "Second", "1.0.0", "",
		catalog.Event[orderCreated](`x"y`, catalog.Sends, catalog.WithSummary("</script><script>alert(3)</script>")))

	exp := architecture.NewExporter(architecture.WithGeneratedAt(excludeFrozenAt()))

	out, err := exp.Export(builder.Build())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	html := string(out)

	for _, forbidden := range []string{
		"<script>alert(1)</script>",
		`" onload="alert(2)`,
		"</script><script>alert(3)</script>",
	} {
		if strings.Contains(html, forbidden) {
			t.Errorf("unescaped catalog data reached the output: %q", forbidden)
		}
	}

	if !strings.Contains(html, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("expected the hostile name to appear escaped")
	}
}

func TestExportIsSelfContained(t *testing.T) {
	out := renderFixture(t)

	for _, forbidden := range []string{
		`src="http`,
		`href="http`,
		"<link ",
		"@import",
	} {
		if strings.Contains(out, forbidden) {
			t.Errorf("document is not self-contained: found %q", forbidden)
		}
	}
}

func TestExportFileWritesDocument(t *testing.T) {
	path := t.TempDir() + "/architecture.html"

	if err := fixtureExporter().ExportFile(fixtureCatalog(), path); err != nil {
		t.Fatalf("ExportFile: %v", err)
	}

	out := renderFixture(t)

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(written) != out {
		t.Error("ExportFile and Export disagree on the rendered document")
	}
}

func TestExportHonoursOptions(t *testing.T) {
	exp := architecture.NewExporter(
		architecture.WithTitle("Renamed"),
		architecture.WithCommand("make architecture"),
		architecture.WithGeneratedAt(excludeFrozenAt()),
	)

	out, err := exp.Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	html := string(out)

	for _, want := range []string{"Renamed", "make architecture"} {
		if !strings.Contains(html, want) {
			t.Errorf("option was not applied: missing %q", want)
		}
	}
}
