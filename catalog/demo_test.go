package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestMarkDemo_StampsEveryResourceKind(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Demo", "1.0.0")
	reg.MarkDemo()
	reg.AddService(catalog.Service{
		ID: "orders-svc", Name: "Orders", Version: "1.0.0",
	})
	reg.AddCommand("orders-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "place-order", Name: "Place Order", Version: "1.0.0",
	})
	reg.AddEvent("orders-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "order.placed", Name: "Order Placed", Version: "1.0.0",
	})
	reg.AddQuery("orders-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "get-order", Name: "Get Order", Version: "1.0.0",
	})
	reg.AddDomain(catalog.Domain{ID: "ordering", Name: "Ordering", Version: "1.0.0"})
	reg.AddChannel(catalog.Channel{ID: "orders-topic", Name: "Orders Topic", Version: "1.0.0"})
	reg.AddDataStore(catalog.DataStore{ID: "orders-db", Name: "Orders DB", Version: "1.0.0"})
	reg.AddFlow(catalog.Flow{ID: "order-lifecycle", Name: "Order Lifecycle", Version: "1.0.0"})
	reg.AddEntity(catalog.Entity{ID: "order", Name: "Order", Version: "1.0.0"})
	reg.AddDataProduct(
		catalog.DataProduct{ID: "order-facts", Name: "Order Facts", Version: "1.0.0"},
	)
	reg.AddAgent(catalog.Agent{ID: "triage-bot", Name: "Triage Bot", Version: "1.0.0"})
	reg.AddCustomDoc(catalog.CustomDoc{ID: "demo-notes", Title: "Demo Notes", Summary: "Notes"})
	reg.AddTeam(catalog.Team{ID: "orders-team", Name: "Orders Team"})
	reg.AddUser(catalog.User{ID: "lars", Name: "Lars Artmann"})

	cat := reg.Build()

	hasDemoBadge := func(kind, id string, badges []catalog.Badge) {
		t.Helper()
		for _, b := range badges {
			if b.Content == "Demo" {
				return
			}
		}
		t.Errorf("%s %s: no Demo badge, got %v", kind, id, badges)
	}

	hasDemoBadge("service", "orders-svc", cat.Services[0].Badges)
	hasDemoBadge("command", "place-order", cat.Services[0].Commands[0].Badges)
	hasDemoBadge("event", "order.placed", cat.Services[0].Events[0].Badges)
	hasDemoBadge("query", "get-order", cat.Services[0].Queries[0].Badges)
	hasDemoBadge("domain", "ordering", cat.Domains[0].Badges)
	hasDemoBadge("channel", "orders-topic", cat.Channels[0].Badges)
	hasDemoBadge("data store", "orders-db", cat.DataStores[0].Badges)
	hasDemoBadge("flow", "order-lifecycle", cat.Flows[0].Badges)
	hasDemoBadge("entity", "order", cat.Entities[0].Badges)
	hasDemoBadge("data product", "order-facts", cat.DataProducts[0].Badges)
	hasDemoBadge("agent", "triage-bot", cat.Agents[0].Badges)
	hasDemoBadge("custom doc", "demo-notes", cat.CustomDocs[0].Badges)

	if src := cat.Teams[0].Source; src == nil || src.Provider != "demo" {
		t.Errorf("team orders-team: source = %+v, want provider demo", cat.Teams[0].Source)
	}

	if src := cat.Users[0].Source; src == nil || src.Provider != "demo" {
		t.Errorf("user lars: source = %+v, want provider demo", cat.Users[0].Source)
	}
}

func TestMarkDemo_KeepsExplicitBadgesAndSource(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Demo", "1.0.0")
	reg.MarkDemo()
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Svc", Version: "1.0.0",
		Badges: []catalog.Badge{{Content: "Internal", BackgroundColor: "blue", TextColor: "white"}},
	})
	reg.AddTeam(catalog.Team{
		ID: "real-team", Name: "Real Team",
		Source: &catalog.Source{Provider: "github", ID: "lars"},
	})

	cat := reg.Build()

	badges := cat.Services[0].Badges
	if len(badges) != 2 || badges[0].Content != "Internal" || badges[1].Content != "Demo" {
		t.Errorf("badges = %v, want explicit Internal first, Demo appended", badges)
	}

	if src := cat.Teams[0].Source; src == nil || src.Provider != "github" {
		t.Errorf("team source = %+v, want explicit github source preserved", cat.Teams[0].Source)
	}
}

func TestMarkDemo_DoesNotDuplicateBadgeAcrossBuilds(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Demo", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "svc",
		Name:    "Svc",
		Version: "1.0.0",
		Badges: []catalog.Badge{
			{Content: "Demo", BackgroundColor: "#f59e0b", TextColor: "#1f2937"},
		},
	})
	reg.MarkDemo()

	for build := range 2 {
		badges := reg.Build().Services[0].Badges
		if len(badges) != 1 {
			t.Errorf(
				"build %d: badges = %v, want the single existing Demo badge untouched",
				build,
				badges,
			)
		}
	}
}

func TestBuilder_MarkDemo(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Demo", "1.0.0")
	b.MarkDemo()
	b.AddUser(catalog.User{ID: "lars", Name: "Lars Artmann"})

	cat := b.Build()
	if src := cat.Users[0].Source; src == nil || src.Provider != "demo" {
		t.Errorf("user source = %+v, want provider demo via Builder.MarkDemo", cat.Users[0].Source)
	}
}
