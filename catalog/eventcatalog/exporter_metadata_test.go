package eventcatalog

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/internal/cattest"
)

func TestExporter_Export_YAMLFrontmatter(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	cattest.AddServiceWithSummary(t, reg, "svc", "Service", "2.0.0", "A test service")

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "services", "svc", "index.mdx")
	if !strings.HasPrefix(content, "---\n") {
		t.Error("MDX file should start with YAML frontmatter")
	}

	if !strings.Contains(content, "---\n\n# ") {
		t.Error("MDX file should have content after frontmatter with heading")
	}
}

func TestExporter_Export_CommandsAndQueriesInServiceFrontmatter(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("order-svc"), "Order Service", "1.0.0")
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
	})
	cattest.AddServiceWithQuery(
		t,
		reg,
		catalog.ServiceID("order-svc"),
		catalog.MessageID("GetOrder"),
		"GetOrder",
		"1.0.0",
		"",
	)

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(t, tmpDir, "services", "order-svc", "index.mdx"),
		"service file",
		"commands:",
		"- id: CreateOrder",
		"queries:",
		"- id: GetOrder",
	)
}

func TestExporter_Export_LLMsTxt(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("MyCatalog", "1.0.0")
	cattest.AddServiceWithSummary(t, reg, "order-svc", "Order Service", "1.0.0", "Manages orders")
	cattest.AddCreateOrderCommand(t, reg, "CreateOrder")
	cattest.AddEventWithSummary(
		t,
		reg,
		catalog.ServiceID("order-svc"),
		catalog.MessageID("OrderCreated"),
		"OrderCreated",
		"1.0.0",
		"Order was created",
		catalog.Sends,
	)
	cattest.AddMessageSimple(
		t,
		reg,
		catalog.ServiceID("order-svc"),
		catalog.MessageID("GetOrder"),
		"GetOrder",
		"1.0.0",
		"Get order by ID",
		catalog.QueryMessage, reg.AddQuery,
	)

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(t, tmpDir, "llms.txt"),
		"llms.txt",
		"# MyCatalog",
		"## Order Service (order-svc)",
		"### Commands",
		"- CreateOrder (v1.0.0): Create a new order",
		"### Events",
		"[sends]",
		"### Queries",
		"- GetOrder (v1.0.0): Get order by ID",
	)
}

func TestExporter_Export_ExamplesFile(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc"), "Svc", "1.0.0")
	cattest.AddCommandWithExample(
		t, reg, catalog.MessageID("CreateOrder"), "CreateOrder", "1.0.0",
		`{"orderId":"abc-123","amount":42.5}`,
	)

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(t, tmpDir, "services", "svc", "commands", "CreateOrder", "examples.json"),
		"examples.json",
		"orderId",
		"42.5",
	)
}

func TestExporter_Export_ServiceWithOwners(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "Test", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "owned-svc",
		Name:    "Owned Service",
		Version: "1.0.0",
		Owners:  []string{"team-platform", "john-doe"},
		Commands: []catalog.Message{
			{Kind: catalog.CommandMessage, ID: "DoThing", Name: "DoThing", Version: "1.0.0"},
		},
	})

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(t, tmpDir, "services", "owned-svc", "index.mdx"),
		"service index.mdx",
		"owners:",
		"- team-platform",
		"- john-doe",
	)
}

func TestExporter_Export_MessageWithoutSummary(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "Test", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc"), "Service", "1.0.0")
	cattest.AddEvent(
		t,
		reg,
		catalog.ServiceID("svc"),
		catalog.MessageID("PlainEvent"),
		"PlainEvent",
		"1.0.0",
		catalog.Sends,
	)

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "services", "svc", "events", "PlainEvent", "index.mdx")

	if strings.Contains(content, "summary:") {
		t.Errorf("message without summary should not have summary field, got:\n%s", content)
	}

	if !strings.Contains(content, "# PlainEvent") {
		t.Errorf("message should have title heading, got:\n%s", content)
	}
}
