package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest"
)

func TestRegistry_AddDataStore(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders DB", Version: "1.0.0",
		ContainerType: "database", Technology: "postgres@16",
	})

	cat := reg.Build()
	if len(cat.DataStores) != 1 {
		t.Fatalf("expected 1 data store, got %d", len(cat.DataStores))
	}

	ds := cat.DataStores[0]
	if ds.ID != "orders-db" {
		t.Errorf("expected orders-db, got %s", ds.ID)
	}

	if ds.ContainerType != "database" {
		t.Errorf("expected database, got %s", ds.ContainerType)
	}
}

func TestRegistry_AddFlow(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddFlow(cattest.NewTestCreateOrderFlow("Submit"))

	cat := reg.Build()
	if len(cat.Flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(cat.Flows))
	}

	f := cat.Flows[0]
	if f.ID != "create-order" {
		t.Errorf("expected create-order, got %s", f.ID)
	}

	if len(f.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(f.Steps))
	}
}

func TestRegistry_AddTeam(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddTeam(catalog.Team{
		ID: "order-team", Name: "Order Team",
		Members: []string{"alice", "bob"},
	})

	cat := reg.Build()
	if len(cat.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(cat.Teams))
	}

	team := cat.Teams[0]
	if team.ID != "order-team" {
		t.Errorf("expected order-team, got %s", team.ID)
	}

	if len(team.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(team.Members))
	}
}

func TestRegistry_AddUser(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddUser(catalog.User{
		ID: "alice", Name: "Alice Smith", Role: "Engineer",
	})

	cat := reg.Build()
	if len(cat.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(cat.Users))
	}

	user := cat.Users[0]
	if user.ID != "alice" {
		t.Errorf("expected alice, got %s", user.ID)
	}
}

func TestRegistry_Build_Immutability(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	cattest.AddDataStore(t, reg, "db1", "DB1", "1.0.0", "database")
	reg.AddFlow(catalog.Flow{ID: "f1", Name: "F1", Version: "1.0.0"})
	reg.AddTeam(catalog.Team{ID: "t1", Name: "T1"})
	reg.AddUser(catalog.User{ID: "u1", Name: "U1"})

	cat1 := reg.Build()

	cattest.AddDataStore(t, reg, "db2", "DB2", "1.0.0", "cache")
	reg.AddFlow(catalog.Flow{ID: "f2", Name: "F2", Version: "1.0.0"})

	cat2 := reg.Build()

	if len(cat1.DataStores) != 1 {
		t.Errorf("cat1 should be immutable, expected 1 data store, got %d", len(cat1.DataStores))
	}

	if len(cat2.DataStores) != 2 {
		t.Errorf("cat2 should have 2 data stores, got %d", len(cat2.DataStores))
	}

	if len(cat1.Flows) != 1 {
		t.Errorf("cat1 should be immutable, expected 1 flow, got %d", len(cat1.Flows))
	}

	if len(cat2.Flows) != 2 {
		t.Errorf("cat2 should have 2 flows, got %d", len(cat2.Flows))
	}
}
