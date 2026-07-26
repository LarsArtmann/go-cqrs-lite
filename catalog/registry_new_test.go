package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestRegistry_AddEntity(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
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

	reg := cattest.NewTestRegistry()
	reg.AddDataProduct(catalog.DataProduct{
		ID: "metrics", Name: "Metrics", Version: "1.0.0",
		Inputs:  []catalog.Ref{{ID: "OrderCreated"}},
		Outputs: []catalog.DataProductOutput{{Ref: catalog.Ref{ID: "MetricsReady"}}},
	})

	cat := reg.Build()
	if len(cat.DataProducts) != 1 {
		t.Fatalf("expected 1 data product, got %d", len(cat.DataProducts))
	}

	if len(cat.DataProducts[0].Inputs) != 1 {
		t.Errorf("expected 1 input, got %d", len(cat.DataProducts[0].Inputs))
	}
}

func TestRegistry_AddAgent(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddAgent(catalog.Agent{
		ID: "bot", Name: "Bot", Version: "1.0.0",
		ReadsFrom: []catalog.DataStoreID{"store1"},
		WritesTo:  []catalog.DataStoreID{"store2"},
		Sends:     []catalog.Ref{{ID: "Decision"}},
		Receives:  []catalog.Ref{{ID: "Request"}},
		Model: &catalog.AgentModel{
			Provider: "OpenAI", Name: "gpt-4", Version: "turbo",
		},
	})

	cat := reg.Build()
	if len(cat.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cat.Agents))
	}

	a := cat.Agents[0]
	if len(a.ReadsFrom) != 1 {
		t.Errorf("expected 1 readsFrom, got %d", len(a.ReadsFrom))
	}

	if a.Model == nil || a.Model.Provider != "OpenAI" {
		t.Error("agent model not preserved")
	}
}

func TestBuilder_AddEntity(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddEntity(catalog.Entity{ID: "e1", Name: "Entity1", Version: "1.0.0"})

	cat := b.Build()
	if len(cat.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(cat.Entities))
	}
}

func TestBuilder_AddDataProduct(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDataProduct(catalog.DataProduct{ID: "dp1", Name: "DP1", Version: "1.0.0"})

	cat := b.Build()
	if len(cat.DataProducts) != 1 {
		t.Fatalf("expected 1 data product, got %d", len(cat.DataProducts))
	}
}

func TestBuilder_AddAgent(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddAgent(catalog.Agent{ID: "a1", Name: "Agent1", Version: "1.0.0"})

	cat := b.Build()
	if len(cat.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cat.Agents))
	}
}

func TestDomainOption_UbiquitousLanguage(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
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

	reg := cattest.NewTestRegistry()
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

	reg := cattest.NewTestRegistry()
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

	reg := cattest.NewTestRegistry(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.SetServiceOptions("svc", catalog.ServiceExternalSystem())

	cat := reg.Build()
	if !cat.Services[0].ExternalSystem {
		t.Error("expected ExternalSystem=true")
	}
}

func TestRegistry_NewResourcesSortedByID(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
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

	reg := cattest.NewTestRegistry()
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

func TestEntity_DeepCopyProperties(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	original := catalog.Entity{
		ID:            "order",
		Name:          "Order",
		Version:       "1.0.0",
		AggregateRoot: true,
		Identifier:    "id",
		Properties: []catalog.EntityProperty{
			{Name: "id", Type: "string", Required: true},
			{Name: "customerId", Type: "string", References: "Customer"},
		},
	}
	reg.AddEntity(original)

	original.Properties[0].Name = "mutated"

	cat := reg.Build()
	if cat.Entities[0].Properties[0].Name != "id" {
		t.Error("entity properties were not deep-copied on Add")
	}

	if !cat.Entities[0].AggregateRoot {
		t.Error("AggregateRoot not preserved")
	}

	if cat.Entities[0].Identifier != "id" {
		t.Error("Identifier not preserved")
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
