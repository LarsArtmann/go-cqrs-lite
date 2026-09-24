// Command ec-fixture exports a fixed demo catalog with the EventCatalog
// exporter. It backs `nix run .#check-eventcatalog`: the render-validation
// flow (generate → npm install → eventcatalog build) needs a stable fixture
// that exercises EVERY resource kind EventCatalog renders and every
// frontmatter field class known to have caused schema-validation failures
// (channel message pointers, changelog/labels/responses x- mapping,
// ubiquitous language, entities/flows pointers, badges, flow step node
// kinds, team x- fields, custom docs).
//
// Profiles (optional second arg):
//   - changelog: omits agents so the generated config can enable changelog
//     pages (@eventcatalog/core 4.6.3 crashes on agent changelog pages —
//     see shouldEnableChangelog)
//   - plain: same resources as the default profile but exported with
//     WithPlainRefIDs — the @eventcatalog/linter (frontmatter-ID keyed)
//     variant the check-eventcatalog gate lints with refs/resource-exists
//     and best-practices/owner-required fully re-armed.
//
// Usage: ec-fixture <output-dir> [changelog|plain]
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: ec-fixture <output-dir> [changelog|plain]")
		os.Exit(2)
	}

	profile := ""
	if len(os.Args) == 3 {
		profile = os.Args[2]
	}

	if profile != "" && profile != "changelog" && profile != "plain" {
		fmt.Fprintf(os.Stderr, "ec-fixture: unknown profile %q (want changelog|plain)\n", profile)
		os.Exit(2)
	}

	if err := run(os.Args[1], profile == "changelog", profile == "plain"); err != nil {
		fmt.Fprintf(os.Stderr, "ec-fixture: %v\n", err)
		os.Exit(1)
	}
}

const (
	fixtureVersion = "1.0.0"
	// Repeated resource IDs: goconst-clean and a single source of truth for
	// the cross-references between fixture resources.
	fixtureServiceID = "order-svc"
	fixtureTeamID    = "order-team"
	fixtureEventID   = "OrderCreated"
	fixtureFlowID    = "checkout-flow"
)

func run(outputDir string, changelogProfile, plainProfile bool) error {
	reg := catalog.NewRegistry("Demo", fixtureVersion)
	visualiser := true

	reg.AddService(catalog.Service{
		ID:       fixtureServiceID,
		Name:     "Order Service",
		Version:  fixtureVersion,
		Summary:  "Manages orders",
		WritesTo: []catalog.DataStoreID{"orders-db"},
		Owners:   []string{fixtureTeamID},
		Entities: []string{"Order"},
		Flows:    []catalog.FlowID{fixtureFlowID},
		Badges: []catalog.Badge{
			{Content: "tier-1"},
			{Content: "stable", BackgroundColor: "green", TextColor: "white"},
		},
		Repository: &catalog.Repository{
			Language: "go",
			URL:      "https://github.com/example/order-svc",
		},
		BaseConfig: catalog.BaseConfig{
			Sidebar: &catalog.SidebarConfig{Badge: "v1", Label: "Orders"},
			Styles: &catalog.StylesConfig{
				Icon:      "server",
				NodeColor: "blue",
				NodeLabel: "orders",
			},
			Visualiser: &visualiser,
			EditUrl:    "https://github.com/example/order-svc/edit/main/docs",
		},
	})
	reg.AddCommand(fixtureServiceID, catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       "CreateOrder",
		Name:     "Create Order",
		Version:  fixtureVersion,
		Summary:  "Create a new order",
		Labels:   map[string]string{"domain": "ordering"},
		Channels: []catalog.ChannelID{"order-events"},
		Owners:   []string{fixtureTeamID},
		Operation: &catalog.Operation{
			Method:      "POST",
			Path:        "/orders",
			StatusCodes: []string{"201", "400"},
		},
		Responses: []catalog.ResponseSpec{
			{StatusCode: "201", Description: "Order created"},
			{StatusCode: "400", Description: "Validation error"},
		},
		Changelog: []catalog.Change{
			{
				Version: "1.0.0",
				Summary: "initial release",
				Date:    new(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)),
			},
		},
		Examples: []jsontext.Value{jsontext.Value(`{"orderId":"abc-123","amount":42.5}`)},
		Schema: &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{
			"orderId": {Type: catalog.TypeString},
		}},
	})
	reg.AddEvent(fixtureServiceID, catalog.Message{
		Kind: catalog.EventMessage, ID: fixtureEventID, Name: "Order Created",
		Version: fixtureVersion, Summary: "Order was created", Direction: catalog.Sends,
		Channels: []catalog.ChannelID{"order-events"},
		Owners:   []string{fixtureTeamID},
		Deprecation: &catalog.DeprecationInfo{
			Message: "superseded by OrderPlaced",
			Date:    new(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
		},
	})
	reg.AddQuery(fixtureServiceID, catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order",
		Version: fixtureVersion, Summary: "Get order by ID",
		Owners: []string{fixtureTeamID},
	})

	reg.AddChannel(catalog.Channel{
		ID: "order-events", Name: "Order Events", Version: fixtureVersion,
		Summary: "All order-related events", Protocols: []catalog.Protocol{"kafka"},
		Address: "orders.events", DeliveryGuarantee: "at-least-once",
		Messages: []catalog.MessageID{"CreateOrder", fixtureEventID},
		Owners:   []string{fixtureTeamID},
		Parameters: map[string]catalog.ChannelParam{
			"region": {Enum: []string{"eu", "us"}, Default: "eu", Description: "Cluster region"},
		},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders Database", Version: fixtureVersion,
		ContainerType: "database", Technology: "postgres@16",
		Summary: "Primary order store", Classification: "confidential",
		Retention: "7y", Residency: "eu",
		Authoritative: true, AccessMode: "readWrite",
		Owners: []string{fixtureTeamID},
	})

	reg.AddDomain(catalog.Domain{
		ID: "ordering", Name: "Ordering", Version: fixtureVersion,
		Summary: "Order management domain", Services: []catalog.ServiceID{fixtureServiceID},
		Entities:   []string{"Order"},
		Flows:      []catalog.FlowID{fixtureFlowID},
		SubDomains: []catalog.DomainID{"checkout"},
		Owners:     []string{fixtureTeamID},
	})
	reg.AddDomain(catalog.Domain{
		ID: "checkout", Name: "Checkout", Version: fixtureVersion,
		Summary: "Checkout subdomain", Owners: []string{fixtureTeamID},
		UbiquitousLanguage: []catalog.UbiquitousLanguageTerm{
			{Name: "Cart", Description: "Items pending purchase"},
			{Name: "Fulfillment", Description: "The process of completing an order"},
		},
	})

	reg.AddEntity(catalog.Entity{
		ID: "Order", Name: "Order", Version: fixtureVersion,
		Summary: "Order aggregate root", AggregateRoot: true, Identifier: "orderId",
		Properties: []catalog.EntityProperty{
			{
				Name:        "orderId",
				Type:        "string",
				Required:    true,
				Description: "Unique order identifier",
			},
			{Name: "items", Type: "array", RelationType: "one-to-many", References: "OrderItem"},
		},
		Owners: []string{fixtureTeamID},
	})
	reg.AddEntity(catalog.Entity{
		ID: "OrderItem", Name: "Order Item", Version: fixtureVersion,
		Summary: "A line item of an order", Identifier: "itemId",
		Properties: []catalog.EntityProperty{
			{Name: "itemId", Type: "string", Required: true},
			{Name: "quantity", Type: "integer"},
		},
		Owners: []string{fixtureTeamID},
	})

	if !changelogProfile {
		// The default profile covers agents; the changelog profile omits them
		// because @eventcatalog/core 4.6.3 crashes rendering agent changelog
		// pages, which keeps shouldEnableChangelog(false).
		reg.AddAgent(catalog.Agent{
			ID:      "order-bot",
			Name:    "Order Bot",
			Version: fixtureVersion,
			Summary: "AI assistant for order support",
			Owners:  []string{fixtureTeamID},
			Sends:   []catalog.Ref{{ID: fixtureEventID, Version: fixtureVersion}},
			Model:   &catalog.AgentModel{Provider: "openai", Name: "gpt", Version: "4o"},
			Tools: []catalog.AgentTool{
				{Name: "orders-db-lookup", Type: "mcp", URL: "https://mcp.example.com/orders"},
			},
			Flows: []catalog.FlowID{fixtureFlowID},
		})
	}

	reg.AddDataProduct(catalog.DataProduct{
		ID:      "order-analytics",
		Name:    "Order Analytics",
		Version: fixtureVersion,
		Summary: "Aggregated order metrics for BI",
		Owners:  []string{fixtureTeamID},
		Inputs:  []catalog.Ref{{ID: fixtureEventID, Version: fixtureVersion}},
		Outputs: []catalog.DataProductOutput{
			{
				Ref:      catalog.Ref{ID: fixtureEventID, Version: fixtureVersion},
				Contract: &catalog.DataContract{Path: "contracts/orders.yaml", Name: "orders"},
			},
		},
	})

	reg.AddFlow(catalog.Flow{
		ID: fixtureFlowID, Name: "Checkout Flow", Version: fixtureVersion,
		Summary: "From command to event",
		Owners:  []string{fixtureTeamID},
		Steps:   checkoutSteps(changelogProfile),
	})

	reg.AddTeam(catalog.Team{
		ID:      fixtureTeamID,
		Name:    "Order Team",
		Summary: "Owns ordering", Members: []string{"alice"},
		Email: "orders@example.com", Role: "platform", AvatarURL: "https://example.com/team.png",
	})
	reg.AddUser(catalog.User{
		ID:        "alice",
		Name:      "Alice Smith",
		Role:      "Senior Engineer",
		Email:     "alice@example.com",
		AvatarURL: "https://example.com/alice.png",
	})

	reg.AddCustomDoc(catalog.CustomDoc{
		ID:      "architecture",
		Title:   "Architecture Overview",
		Summary: "How the demo fits together",
		Slug:    "guides/architecture",
		Content: "## Context\nThe demo shows every resource kind.\n## Decision\nGenerate docs from Go types.",
		Owners:  []string{fixtureTeamID},
	})

	exporter := eventcatalog.NewExporter(outputDir)
	if plainProfile {
		exporter = eventcatalog.NewExporter(outputDir, eventcatalog.WithPlainRefIDs())
	}

	if err := exporter.Export(reg.Build()); err != nil {
		return err
	}

	// The data product's output contract points at a contract FILE; the
	// linter's refs/file-exists rule (and EventCatalog's contract rendering)
	// expects it inside the data product's directory. The exporter copies
	// nothing (contract files are the declaring repo's assets) — the fixture
	// writes its own.
	contractDir := filepath.Join(outputDir, "data-products", "order-analytics", "contracts")
	if err := os.MkdirAll(contractDir, 0o750); err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(contractDir, "orders.yaml"),
		[]byte(fixtureDataContract),
		0o600,
	)
}

// fixtureDataContract is the data product output contract the fixture
// places at contracts/orders.yaml.
const fixtureDataContract = `# Data contract: orders (fixture)
id: orders
owner: order-team
type: table
description: One row per order event aggregate
fields:
  - name: order_id
    type: string
    required: true
`

func checkoutSteps(withAgent bool) []catalog.FlowStep {
	steps := []catalog.FlowStep{
		{
			ID:    "s1",
			Title: "Customer",
			Actor: &catalog.FlowActor{Name: "Customer", Summary: "Places orders"},
		},
		{
			ID:       "s2",
			Title:    "Submit",
			Service:  &catalog.FlowStepRef{ID: fixtureServiceID},
			NextStep: &catalog.FlowEdge{ID: "s3"},
		},
		{
			ID:        "s3",
			Title:     "Create",
			Message:   &catalog.FlowStepRef{ID: "CreateOrder"},
			NextSteps: []catalog.FlowEdge{{ID: "s4"}},
		},
		{ID: "s4", Title: "Persist", DataStore: &catalog.FlowStepRef{ID: "orders-db"}},
	}
	if withAgent {
		steps = append(
			steps,
			catalog.FlowStep{
				ID:    "s5",
				Title: "AI summary",
				Agent: &catalog.FlowStepRef{ID: "order-bot"},
			},
		)
	}

	return append(
		steps,
		catalog.FlowStep{
			ID:          "s6",
			Title:       "Analytics",
			DataProduct: &catalog.FlowStepRef{ID: "order-analytics"},
		},
		catalog.FlowStep{
			ID:       "s7",
			Title:    "Payment provider",
			External: &catalog.FlowActor{Name: "Stripe", URL: "https://stripe.com"},
		},
		catalog.FlowStep{
			ID:     "s8",
			Title:  "Audit",
			Custom: &catalog.FlowCustomNode{Title: "Audit log", Icon: "clipboard"},
		},
	)
}
