package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestBuilder_AddDataStore(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders DB", Version: "1.0.0",
		ContainerType: "database", Technology: "postgres@16",
	})

	cat := b.Build()
	if len(cat.DataStores) != 1 {
		t.Fatalf("expected 1 data store, got %d", len(cat.DataStores))
	}

	if cat.DataStores[0].Technology != "postgres@16" {
		t.Errorf("expected postgres@16, got %s", cat.DataStores[0].Technology)
	}
}

func TestBuilder_AddFlow(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddFlow(catalog.Flow{
		ID: "create-order", Name: "Create Order", Version: "1.0.0",
		Steps: []catalog.FlowStep{
			{ID: "1", Title: "Submit", Message: &catalog.FlowStepRef{ID: "CreateOrder"}},
			{ID: "2", Title: "Process", Service: &catalog.FlowStepRef{ID: "order-svc"}},
		},
	})

	cat := b.Build()
	if len(cat.Flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(cat.Flows))
	}

	if len(cat.Flows[0].Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(cat.Flows[0].Steps))
	}
}

func TestBuilder_AddChannel(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddChannel(catalog.Channel{
		ID: "order-events", Name: "Order Events", Version: "1.0.0",
		Protocols: []catalog.Protocol{"kafka"}, Address: "orders.events",
	})

	cat := b.Build()
	if len(cat.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(cat.Channels))
	}

	if cat.Channels[0].Address != "orders.events" {
		t.Errorf("expected orders.events, got %s", cat.Channels[0].Address)
	}
}

func TestBuilder_AddTeam(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddTeam(catalog.Team{ID: "order-team", Name: "Order Team", Members: []string{"alice"}})

	cat := b.Build()
	if len(cat.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(cat.Teams))
	}
}

func TestBuilder_AddUser(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddUser(catalog.User{ID: "alice", Name: "Alice", Role: "Engineer"})

	cat := b.Build()
	if len(cat.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(cat.Users))
	}
}

func TestBuilder_AllResourceTypes(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Full Catalog", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "Test service",
		catalog.Event[struct{}]("evt", catalog.Sends),
	)
	b.AddDomain("dom", "Domain", "1.0.0", "Test domain", "svc")
	b.AddChannel(catalog.Channel{ID: "ch", Name: "Channel", Version: "1.0.0"})
	b.AddDataStore(
		catalog.DataStore{ID: "ds", Name: "Store", Version: "1.0.0", ContainerType: "database"},
	)
	b.AddFlow(catalog.Flow{ID: "fl", Name: "Flow", Version: "1.0.0"})
	b.AddTeam(catalog.Team{ID: "tm", Name: "Team"})
	b.AddUser(catalog.User{ID: "us", Name: "User"})

	cat := b.Build()

	assertCount(t, "services", len(cat.Services), 1)
	assertCount(t, "domains", len(cat.Domains), 1)
	assertCount(t, "channels", len(cat.Channels), 1)
	assertCount(t, "dataStores", len(cat.DataStores), 1)
	assertCount(t, "flows", len(cat.Flows), 1)
	assertCount(t, "teams", len(cat.Teams), 1)
	assertCount(t, "users", len(cat.Users), 1)
}

func assertCount(t *testing.T, name string, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("expected %d %s, got %d", want, name, got)
	}
}
