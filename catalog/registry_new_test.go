package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func TestRegistry_AddEntity(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddEntity(catalog.Entity{
		ID: "order", Name: "Order", Version: "1.0.0",
		Summary: "Order aggregate", Owners: []string{"team-a"},
	})

	cat := reg.Build()
	if len(cat.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(cat.Entities))
	}

	if cat.Entities[0].ID != "order" {
		t.Errorf("entity ID = %s, want order", cat.Entities[0].ID)
	}
}

func TestRegistry_AddDataProduct(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddDataProduct(catalog.DataProduct{
		ID: "metrics", Name: "Metrics", Version: "1.0.0", Domain: "analytics",
	})

	cat := reg.Build()
	if len(cat.DataProducts) != 1 {
		t.Fatalf("expected 1 data product, got %d", len(cat.DataProducts))
	}

	if cat.DataProducts[0].Domain != "analytics" {
		t.Errorf("data product domain = %s, want analytics", cat.DataProducts[0].Domain)
	}
}

func TestRegistry_AddAgent(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddAgent(catalog.Agent{
		ID: "bot", Name: "Bot", Version: "1.0.0",
		DataStores: []catalog.DataStoreID{"store1"},
	})

	cat := reg.Build()
	if len(cat.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cat.Agents))
	}

	if len(cat.Agents[0].DataStores) != 1 {
		t.Errorf("expected 1 data store on agent, got %d", len(cat.Agents[0].DataStores))
	}
}

func TestBuilder_AddEntity(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddEntity(catalog.Entity{ID: "e1", Name: "Entity1", Version: "1.0.0"})

	cat := b.Build()
	if len(cat.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(cat.Entities))
	}
}

func TestBuilder_AddDataProduct(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddDataProduct(catalog.DataProduct{ID: "dp1", Name: "DP1", Version: "1.0.0"})

	cat := b.Build()
	if len(cat.DataProducts) != 1 {
		t.Fatalf("expected 1 data product, got %d", len(cat.DataProducts))
	}
}

func TestBuilder_AddAgent(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddAgent(catalog.Agent{ID: "a1", Name: "Agent1", Version: "1.0.0"})

	cat := b.Build()
	if len(cat.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cat.Agents))
	}
}

func TestDomainOption_UbiquitousLanguage(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddDomain(catalog.Domain{ID: "d1", Name: "D1", Version: "1.0.0"})
	reg.SetDomainOptions(
		"d1",
		catalog.DomainUbiquitousLanguage(
			catalog.UbiquitousLanguageTerm{Name: "Order", Description: "A purchase request"},
			catalog.UbiquitousLanguageTerm{Name: "Cart", Description: "Shopping cart"},
		),
	)

	cat := reg.Build()
	if len(cat.Domains[0].UbiquitousLanguage) != 2 {
		t.Fatalf("expected 2 ubiquitous language terms, got %d",
			len(cat.Domains[0].UbiquitousLanguage))
	}
}

func TestDomainOption_SubDomains(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddDomain(catalog.Domain{ID: "parent", Name: "Parent", Version: "1.0.0"})
	reg.SetDomainOptions(
		"parent",
		catalog.DomainSubDomains("child1", "child2"),
	)

	cat := reg.Build()
	if len(cat.Domains[0].SubDomains) != 2 {
		t.Fatalf("expected 2 sub-domains, got %d", len(cat.Domains[0].SubDomains))
	}
}

func TestDomainOption_DataProducts(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddDomain(catalog.Domain{ID: "d1", Name: "D1", Version: "1.0.0"})
	reg.SetDomainOptions(
		"d1",
		catalog.DomainDataProducts("dp1"),
	)

	cat := reg.Build()
	if len(cat.Domains[0].DataProducts) != 1 {
		t.Fatalf("expected 1 data product ref, got %d", len(cat.Domains[0].DataProducts))
	}
}

func TestServiceOption_ExternalSystem(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.SetServiceOptions("svc", catalog.ServiceExternalSystem())

	cat := reg.Build()
	if !cat.Services[0].ExternalSystem {
		t.Error("expected ExternalSystem=true")
	}
}

func TestRegistry_NewResourcesSortedByID(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	reg.AddEntity(catalog.Entity{ID: "zebra", Name: "Z", Version: "1.0.0"})
	reg.AddEntity(catalog.Entity{ID: "alpha", Name: "A", Version: "1.0.0"})
	reg.AddDataProduct(catalog.DataProduct{ID: "zeta", Name: "Z", Version: "1.0.0"})
	reg.AddDataProduct(catalog.DataProduct{ID: "beta", Name: "B", Version: "1.0.0"})

	cat := reg.Build()
	if cat.Entities[0].ID != "alpha" {
		t.Errorf("entities not sorted: first = %s", cat.Entities[0].ID)
	}

	if cat.DataProducts[0].ID != "beta" {
		t.Errorf("data products not sorted: first = %s", cat.DataProducts[0].ID)
	}
}

func TestRegistry_NewResourcesAreCopied(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test", "1.0.0")
	original := catalog.Entity{
		ID: "e1", Name: "E1", Version: "1.0.0", Owners: []string{"a", "b"},
	}
	reg.AddEntity(original)

	original.Owners[0] = "mutated"

	cat := reg.Build()
	if cat.Entities[0].Owners[0] != "a" {
		t.Error("entity was not deep-copied on Add")
	}
}

func TestFlowStep_NewTypes(t *testing.T) {
	t.Parallel()

	step := catalog.FlowStep{
		ID:          "s1",
		Title:       "Step",
		Agent:       &catalog.FlowStepRef{ID: "bot"},
		DataStore:   &catalog.FlowStepRef{ID: "db"},
		DataProduct: &catalog.FlowStepRef{ID: "dp"},
		SubFlow:     &catalog.FlowStepRef{ID: "child"},
	}

	if step.Agent.ID != "bot" {
		t.Error("Agent ref not set")
	}

	if step.DataStore.ID != "db" {
		t.Error("DataStore ref not set")
	}

	if step.DataProduct.ID != "dp" {
		t.Error("DataProduct ref not set")
	}

	if step.SubFlow.ID != "child" {
		t.Error("SubFlow ref not set")
	}
}
