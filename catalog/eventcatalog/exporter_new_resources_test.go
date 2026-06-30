package eventcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/internal/cattest"
)

func TestExporter_Entity(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddEntity(catalog.Entity{
		ID:      "order-entity",
		Name:    "Order",
		Version: "1.0.0",
		Summary: "The Order aggregate root",
		Owners:  []string{"team-orders"},
		Schema: &catalog.Schema{
			Type: catalog.TypeObject,
			Properties: map[string]catalog.Property{
				"id":     {Type: catalog.TypeString},
				"status": {Type: catalog.TypeString},
			},
		},
	})

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "entities", "order-entity", "index.mdx")
	cattest.AssertContentContains(
		t, content, "entity mdx",
		"id: order-entity",
		"name: Order",
		"summary: \"The Order aggregate root\"",
		"schemaPath: schemas/schema.json",
		"# Order",
	)

	assertFileExists(t, tmpDir, "schema file should exist",
		"entities", "order-entity", "schemas", "schema.json")
}

func TestExporter_EntityWithoutSchema(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddEntity(catalog.Entity{
		ID:      "user-entity",
		Name:    "User",
		Version: "1.0.0",
	})

	tmpDir := exportCatalog(t, reg)
	content := readExported(t, tmpDir, "entities", "user-entity", "index.mdx")

	if strings.Contains(content, "schemaPath") {
		t.Errorf("entity without schema should not have schemaPath")
	}
}

func TestExporter_DataProduct(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddDataProduct(catalog.DataProduct{
		ID:      "order-analytics",
		Name:    "Order Analytics",
		Version: "1.0.0",
		Summary: "Aggregated order metrics for BI",
		Owners:  []string{"data-team"},
		Inputs: []catalog.Ref{
			{ID: "OrderConfirmed", Version: "1.0.0"},
			{ID: "PaymentProcessed"},
		},
		Outputs: []catalog.Ref{
			{ID: "OrderMetricsCalculated", Version: "1.0.0"},
		},
	})

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "data-products", "order-analytics", "index.mdx")
	cattest.AssertContentContains(
		t, content, "data product mdx",
		"id: order-analytics",
		"name: Order Analytics",
		"inputs:",
		"outputs:",
		"id: OrderConfirmed",
		"version: \"1.0.0\"",
	)
}

func TestExporter_Agent(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddAgent(catalog.Agent{
		ID:      "order-bot",
		Name:    "Order Bot",
		Version: "1.0.0",
		Summary: "AI agent that handles order processing",
		Owners:  []string{"ai-team"},
		Sends: []catalog.Ref{
			{ID: "FraudReviewCompleted", Version: "1.0.0"},
		},
		Receives: []catalog.Ref{
			{ID: "PaymentInitiated", Version: "1.0.0"},
		},
		ReadsFrom: []catalog.DataStoreID{"fraud-db"},
		WritesTo:  []catalog.DataStoreID{"audit-db"},
		Model: &catalog.AgentModel{
			Provider: "OpenAI",
			Name:     "gpt-4.1",
			Version:  "2025-04-14",
		},
		Tools: []catalog.AgentTool{
			{
				Name: "Risk lookup", Type: "mcp", URL: "https://mcp.example.com/risk",
				Description: "Retrieves risk signals",
			},
		},
	})

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "agents", "order-bot", "index.mdx")
	cattest.AssertContentContains(
		t, content, "agent mdx",
		"id: order-bot",
		"name: Order Bot",
		"sends:",
		"receives:",
		"readsFrom:",
		"writesTo:",
		"model:",
		"provider: \"OpenAI\"",
		"name: \"gpt-4.1\"",
		"tools:",
		"type: mcp",
	)
}

func TestExporter_AgentMinimal(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddAgent(catalog.Agent{
		ID:      "simple-bot",
		Name:    "Simple Bot",
		Version: "1.0.0",
	})

	tmpDir := exportCatalog(t, reg)
	content := readExported(t, tmpDir, "agents", "simple-bot", "index.mdx")

	if strings.Contains(content, "model:") {
		t.Errorf("agent without model should not have model field")
	}

	if strings.Contains(content, "tools:") {
		t.Errorf("agent without tools should not have tools field")
	}
}

func TestExporter_DomainWithUbiquitousLanguage(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddDomain(catalog.Domain{
		ID:      "orders",
		Name:    "Orders",
		Version: "1.0.0",
		Summary: "Order management domain",
		UbiquitousLanguage: []catalog.UbiquitousLanguageTerm{
			{Name: "Order", Description: "A customer request to purchase items"},
			{Name: "Fulfillment", Description: "The process of completing an order"},
		},
		SubDomains:   []catalog.DomainID{"checkout", "shipping"},
		DataProducts: []catalog.DataProductID{"order-analytics"},
	})

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "domains", "orders", "index.mdx")
	cattest.AssertContentContains(
		t, content, "domain mdx",
		"id: orders",
		"ubiquitousLanguage:",
		"name: \"Order\"",
		"description: \"A customer request to purchase items\"",
		"data-products:",
	)
}

func TestExporter_ServiceExternalSystem(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "stripe", Name: "Stripe", Version: "1.0.0",
		ExternalSystem: true,
	})

	tmpDir := exportCatalog(t, reg)
	content := readExported(t, tmpDir, "services", "stripe", "index.mdx")

	if !strings.Contains(content, "externalSystem: true") {
		t.Errorf("external service should have externalSystem: true\n%s", content)
	}
}

func TestExporter_ServiceNotExternal(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "internal-svc", Name: "Internal", Version: "1.0.0",
	})

	tmpDir := exportCatalog(t, reg)
	content := readExported(t, tmpDir, "services", "internal-svc", "index.mdx")

	if strings.Contains(content, "externalSystem") {
		t.Errorf("internal service should not have externalSystem field")
	}
}

func TestExporter_FlowStepWithNewTypes(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddDataStore(catalog.DataStore{
		ID: "db", Name: "DB", Version: "1.0.0", ContainerType: "database",
	})
	reg.AddDataProduct(catalog.DataProduct{
		ID: "dp1", Name: "DP1", Version: "1.0.0",
	})
	reg.AddAgent(catalog.Agent{
		ID: "bot", Name: "Bot", Version: "1.0.0",
	})
	reg.AddFlow(catalog.Flow{
		ID: "ai-flow", Name: "AI Flow", Version: "1.0.0",
		Steps: []catalog.FlowStep{
			{
				ID: "step-1", Title: "Agent decides",
				Agent:    &catalog.FlowStepRef{ID: "bot"},
				NextStep: &catalog.FlowEdge{ID: "step-2"},
			},
			{
				ID: "step-2", Title: "Read from store",
				DataStore: &catalog.FlowStepRef{ID: "db"},
				NextStep:  &catalog.FlowEdge{ID: "step-3"},
			},
			{
				ID: "step-3", Title: "Produce data",
				DataProduct: &catalog.FlowStepRef{ID: "dp1"},
			},
		},
	})

	tmpDir := exportCatalog(t, reg)
	content := readExported(t, tmpDir, "flows", "ai-flow", "index.mdx")

	cattest.AssertContentContains(
		t, content, "flow with new step types",
		"agent:",
		"dataStore:",
		"dataProduct:",
	)
}

func TestExporter_FlowStepSubFlow(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddFlow(catalog.Flow{
		ID: "parent-flow", Name: "Parent", Version: "1.0.0",
		Steps: []catalog.FlowStep{
			{
				ID: "step-1", Title: "Nested flow",
				SubFlow: &catalog.FlowStepRef{ID: "child-flow"},
			},
		},
	})

	tmpDir := exportCatalog(t, reg)
	content := readExported(t, tmpDir, "flows", "parent-flow", "index.mdx")

	if !strings.Contains(content, "subFlow:") {
		t.Errorf("flow should contain subFlow step\n%s", content)
	}
}

func TestExporter_SchemasTxt(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "DoThing", Name: "DoThing", Version: "1.0.0",
		Schema: &catalog.Schema{
			Type: catalog.TypeObject,
			Properties: map[string]catalog.Property{
				"id": {Type: catalog.TypeString},
			},
		},
	})
	reg.AddEntity(catalog.Entity{
		ID: "thing-entity", Name: "Thing", Version: "1.0.0",
		Schema: &catalog.Schema{
			Type: catalog.TypeObject,
			Properties: map[string]catalog.Property{
				"name": {Type: catalog.TypeString},
			},
		},
	})

	tmpDir := exportCatalog(t, reg)

	schemasPath := filepath.Join(tmpDir, "schemas.txt")
	if _, err := os.Stat(schemasPath); os.IsNotExist(err) {
		t.Fatal("schemas.txt was not generated")
	}

	content := readExported(t, tmpDir, "schemas.txt")
	cattest.AssertContentContains(
		t, content, "schemas.txt",
		"# Schemas",
		"DoThing",
		"Thing",
	)
}

func TestExporter_AllNewResourcesExported(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("FullCatalog", "2.0.0")
	reg.AddService(catalog.Service{ID: "svc1", Name: "Service1", Version: "1.0.0"})
	reg.AddDomain(catalog.Domain{
		ID: "d1", Name: "Domain1", Version: "1.0.0",
		UbiquitousLanguage: []catalog.UbiquitousLanguageTerm{
			{Name: "Term1", Description: "Desc1"},
		},
	})
	reg.AddEntity(catalog.Entity{ID: "e1", Name: "Entity1", Version: "1.0.0"})
	reg.AddDataProduct(catalog.DataProduct{ID: "dp1", Name: "DP1", Version: "1.0.0"})
	reg.AddAgent(catalog.Agent{ID: "a1", Name: "Agent1", Version: "1.0.0"})

	tmpDir := exportCatalog(t, reg)

	for _, path := range []struct {
		parts []string
		desc  string
	}{
		{[]string{"entities", "e1", "index.mdx"}, "entity"},
		{[]string{"data-products", "dp1", "index.mdx"}, "data product"},
		{[]string{"agents", "a1", "index.mdx"}, "agent"},
		{[]string{"domains", "d1", "index.mdx"}, "domain"},
		{[]string{"schemas.txt"}, "schemas.txt"},
		{[]string{"llms.txt"}, "llms.txt"},
	} {
		assertFileExists(t, tmpDir, path.desc+" should exist", path.parts...)
	}
}
