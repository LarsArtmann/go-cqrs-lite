package eventcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

func newCommand(id string) catalog.Message {
	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      catalog.MessageID(id),
		Name:    id,
		Version: "1.0.0",
	}

	return msg
}

func newEvent(id, name string, direction catalog.Direction) catalog.Message {
	return catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        catalog.MessageID(id),
		Name:      name,
		Version:   "1.0.0",
		Direction: direction,
	}
}

func exportCatalog(t *testing.T, reg *catalog.Registry) string {
	t.Helper()
	tmpDir := t.TempDir()
	cat := reg.Build()
	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func readExported(t *testing.T, tmpDir string, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{tmpDir}, parts...)...))
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Join(parts...), err)
	}

	return string(data)
}

func requireExportPermissionError(t *testing.T, cat *catalog.Catalog, tmpDir, readOnlyDir string) {
	t.Helper()

	err := os.MkdirAll(readOnlyDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chmod(readOnlyDir, 0o000)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Chmod(readOnlyDir, 0o750) //nolint:errcheck // cleanup

	exp := NewExporter(tmpDir)

	cattest.RequireErr(t, exp.Export(cat), "expected error when dir is read-only")
}

func TestExporter_Export_ServiceWithCommand(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Summary: "Create a new order",
		Schema:  cattest.StringSchema("orderId", "timestamp"),
	})

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "services", "order-svc", "index.mdx")
	if !strings.Contains(content, "id: order-svc") {
		t.Errorf("service file missing id: %s", content)
	}

	cattest.ReadFileAndAssert(
		t, filepath.Join(tmpDir, "services", "order-svc", "index.mdx"), "service file",
		"name: Order Service",
		"# Order Service",
	)

	cmdContent := readExported(
		t,
		tmpDir,
		"services",
		"order-svc",
		"commands",
		"CreateOrder",
		"index.mdx",
	)
	cattest.AssertContentContains(t, cmdContent, "command file", "id: CreateOrder")

	var schema map[string]any
	schemaData, err := os.ReadFile(
		filepath.Join(
			tmpDir,
			"services",
			"order-svc",
			"commands",
			"CreateOrder",
			"schemas",
			"schema.json",
		),
	)
	if err != nil {
		t.Fatalf("read schema file: %v", err)
	}

	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("parse schema JSON: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
}

func TestExporter_Export_Event(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "payment-svc", Name: "Payment Service", Version: "1.0.0",
	})
	reg.AddEvent("payment-svc", catalog.Message{
		Kind:    catalog.EventMessage,
		ID:      "PaymentCompleted",
		Name:    "PaymentCompleted",
		Version: "1.0.0",
		Summary: "Payment completed successfully",
	})

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(
			t,
			tmpDir,
			"services",
			"payment-svc",
			"events",
			"PaymentCompleted",
			"index.mdx",
		),
		"event file",
		"id: PaymentCompleted",
		"# PaymentCompleted",
	)
}

func TestExporter_Export_Query(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	cattest.AddServiceWithQuery(
		t,
		reg,
		catalog.ServiceID("catalog-svc"),
		catalog.MessageID("GetProduct"),
		"GetProduct",
		"1.0.0",
		"",
	)

	tmpDir := exportCatalog(t, reg)

	if !strings.Contains(
		readExported(t, tmpDir, "services", "catalog-svc", "queries", "GetProduct", "index.mdx"),
		"id: GetProduct",
	) {
		t.Errorf("query file missing id")
	}
}

func TestExporter_Export_Domain(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddDomain(
		t,
		reg,
		catalog.DomainID("ordering"),
		"Ordering",
		"1.0.0",
		"Order management domain",
		[]catalog.ServiceID{"order-svc"},
	)

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(t, tmpDir, "domains", "ordering", "index.mdx"),
		"domain file",
		"id: ordering",
		"services:",
		"- id: order-svc",
	)
}

func TestExporter_Export_Config(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("MyCatalog", "2.0.0")
	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "eventcatalog.config.js")
	if !strings.Contains(content, "title: \"MyCatalog\"") {
		t.Errorf("config missing title: %s", content)
	}
}

func TestExporter_Export_MultipleServices(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc-a", Name: "Service A", Version: "1.0.0"})
	reg.AddService(catalog.Service{ID: "svc-b", Name: "Service B", Version: "1.0.0"})
	reg.AddCommand("svc-a", newCommand("CmdA"))
	reg.AddEvent("svc-b", catalog.Message{
		Kind: catalog.EventMessage, ID: "EvtB", Name: "EvtB", Version: "1.0.0",
	})

	tmpDir := exportCatalog(t, reg)

	if _, err := os.Stat(
		filepath.Join(tmpDir, "services", "svc-a", "index.mdx"),
	); os.IsNotExist(
		err,
	) {
		t.Error("svc-a directory not created")
	}

	if _, err := os.Stat(
		filepath.Join(tmpDir, "services", "svc-b", "index.mdx"),
	); os.IsNotExist(
		err,
	) {
		t.Error("svc-b directory not created")
	}

	if _, err := os.Stat(
		filepath.Join(tmpDir, "services", "svc-a", "commands", "CmdA", "index.mdx"),
	); os.IsNotExist(
		err,
	) {
		t.Error("CmdA command file not created")
	}

	if _, err := os.Stat(
		filepath.Join(tmpDir, "services", "svc-b", "events", "EvtB", "index.mdx"),
	); os.IsNotExist(
		err,
	) {
		t.Error("EvtB event file not created")
	}
}

func TestExporter_Export_NoSchema(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", newCommand("NoSchema"))

	tmpDir := exportCatalog(t, reg)

	_, err := os.Stat(
		filepath.Join(tmpDir, "services", "svc", "commands", "NoSchema", "schemas", "schema.json"),
	)
	if !os.IsNotExist(err) {
		t.Error("schema.json should not exist when no schema is provided")
	}
}

func TestExporter_Export_SchemaPathInFrontmatter(t *testing.T) {
	t.Parallel()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc"), "Svc", "1.0.0")
	cattest.AddCommandWithSchema(
		t,
		reg,
		catalog.ServiceID("svc"),
		catalog.MessageID("CreateOrder"),
		"CreateOrder",
		"1.0.0",
		&catalog.Schema{Type: catalog.TypeObject},
	)

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "services", "svc", "commands", "CreateOrder", "index.mdx")
	if !strings.Contains(content, "schemaPath: schemas/schema.json") {
		t.Errorf("message frontmatter missing schemaPath: %s", content)
	}
}

func TestExporter_Export_NoSchemaPathWhenNoSchema(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "NoSchema",
		Name:    "NoSchema",
		Version: "1.0.0",
	})

	tmpDir := exportCatalog(t, reg)

	if strings.Contains(
		readExported(t, tmpDir, "services", "svc", "commands", "NoSchema", "index.mdx"),
		"schemaPath",
	) {
		t.Error("schemaPath should not appear when no schema provided")
	}
}

func TestExporter_Export_ServiceSendsReceives(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddEvent("order-svc", newEvent("OrderCreated", "OrderCreated", catalog.Sends))
	reg.AddEvent("order-svc", newEvent("PaymentProcessed", "PaymentProcessed", catalog.Receives))

	tmpDir := exportCatalog(t, reg)

	cattest.AssertContentContains(
		t,
		readExported(t, tmpDir, "services", "order-svc", "index.mdx"),
		"service file",
		"id: order-svc",
		"sends:",
		"- id: OrderCreated",
		"receives:",
		"- id: PaymentProcessed",
	)
}

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
	cattest.AddQuerySimple(
		t,
		reg,
		catalog.ServiceID("order-svc"),
		catalog.MessageID("GetOrder"),
		"GetOrder",
		"1.0.0",
		"Get order by ID",
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
		t, reg, "CreateOrder", "CreateOrder", "1.0.0",
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
	cattest.AddEvent(t, reg, "svc", "PlainEvent", "PlainEvent", "1.0.0", catalog.Sends)

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "services", "svc", "events", "PlainEvent", "index.mdx")

	if strings.Contains(content, "summary:") {
		t.Errorf("message without summary should not have summary field, got:\n%s", content)
	}

	if !strings.Contains(content, "# PlainEvent") {
		t.Errorf("message should have title heading, got:\n%s", content)
	}
}
