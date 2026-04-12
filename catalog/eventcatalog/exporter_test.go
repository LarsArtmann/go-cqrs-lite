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
		ID:      id,
		Name:    id,
		Version: "1.0.0",
	}

	return msg
}

func TestExporter_Export_ServiceWithCommand(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

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
		Schema: &catalog.Schema{
			Type: "object",
			Properties: map[string]catalog.Property{
				"orderId":   {Type: "string"},
				"timestamp": {Type: "string"},
			},
		},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	svcPath := filepath.Join(tmpDir, "services", "order-svc", "index.mdx")

	data, err := os.ReadFile(svcPath)
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "id: order-svc") {
		t.Errorf("service file missing id: %s", content)
	}

	cattest.ReadFileAndAssert(t, svcPath, "service file",
		"name: Order Service",
		"# Order Service",
	)

	cmdPath := filepath.Join(
		tmpDir,
		"services",
		"order-svc",
		"commands",
		"CreateOrder",
		"index.mdx",
	)

	data, err = os.ReadFile(cmdPath)
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}

	content = string(data)
	cattest.AssertContentContains(t, content, "command file",
		"id: CreateOrder",
		"owners:",
	)

	schemaPath := filepath.Join(
		tmpDir,
		"services",
		"order-svc",
		"commands",
		"CreateOrder",
		"schemas",
		"schema.json",
	)

	data, err = os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema file: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse schema JSON: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
}

func TestExporter_Export_Event(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

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

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	evtPath := filepath.Join(
		tmpDir,
		"services",
		"payment-svc",
		"events",
		"PaymentCompleted",
		"index.mdx",
	)

	data, err := os.ReadFile(evtPath)
	if err != nil {
		t.Fatalf("read event file: %v", err)
	}

	content := string(data)
	cattest.AssertContentContains(t, content, "event file",
		"id: PaymentCompleted",
		"# PaymentCompleted",
	)
}

func TestExporter_Export_Query(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	cattest.AddService(t, reg, "catalog-svc", "Catalog Service", "1.0.0")
	cattest.AddMessageSimple(t, reg, "catalog-svc", "GetProduct", "GetProduct", "1.0.0", "", catalog.QueryMessage, reg.AddQuery)

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	qPath := filepath.Join(tmpDir, "services", "catalog-svc", "queries", "GetProduct", "index.mdx")

	data, err := os.ReadFile(qPath)
	if err != nil {
		t.Fatalf("read query file: %v", err)
	}

	if !strings.Contains(string(data), "id: GetProduct") {
		t.Errorf("query file missing id")
	}
}

func TestExporter_Export_Domain(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddDomain(
		t,
		reg,
		"ordering",
		"Ordering",
		"1.0.0",
		"Order management domain",
		[]string{"order-svc"},
	)

	cat := cattest.Build(t, reg)

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	domainPath := filepath.Join(tmpDir, "domains", "ordering", "index.mdx")

	data, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read domain file: %v", err)
	}

	content := string(data)
	cattest.AssertContentContains(t, content, "domain file",
		"id: ordering",
		"services:",
		"- order-svc",
	)
}

func TestExporter_Export_Config(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("MyCatalog", "2.0.0")
	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(tmpDir, "eventcatalog.config.js")

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: \"MyCatalog\"") {
		t.Errorf("config missing title: %s", content)
	}
}

func TestExporter_Export_MultipleServices(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc-a", Name: "Service A", Version: "1.0.0"})
	reg.AddService(catalog.Service{ID: "svc-b", Name: "Service B", Version: "1.0.0"})
	reg.AddCommand("svc-a", newCommand("CmdA"))
	reg.AddEvent("svc-b", catalog.Message{
		Kind: catalog.EventMessage, ID: "EvtB", Name: "EvtB", Version: "1.0.0",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)

	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	svcA := filepath.Join(tmpDir, "services", "svc-a", "index.mdx")
	if _, err := os.Stat(svcA); os.IsNotExist(err) {
		t.Error("svc-a directory not created")
	}

	svcB := filepath.Join(tmpDir, "services", "svc-b", "index.mdx")
	if _, err := os.Stat(svcB); os.IsNotExist(err) {
		t.Error("svc-b directory not created")
	}

	cmdA := filepath.Join(tmpDir, "services", "svc-a", "commands", "CmdA", "index.mdx")
	if _, err := os.Stat(cmdA); os.IsNotExist(err) {
		t.Error("CmdA command file not created")
	}

	evtB := filepath.Join(tmpDir, "services", "svc-b", "events", "EvtB", "index.mdx")
	if _, err := os.Stat(evtB); os.IsNotExist(err) {
		t.Error("EvtB event file not created")
	}
}

func TestExporter_Export_NoSchema(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", newCommand("NoSchema"))

	cat := reg.Build()

	exp := NewExporter(tmpDir)

	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	schemaPath := filepath.Join(
		tmpDir,
		"services",
		"svc",
		"commands",
		"NoSchema",
		"schemas",
		"schema.json",
	)
	if _, err := os.Stat(schemaPath); !os.IsNotExist(err) {
		t.Error("schema.json should not exist when no schema is provided")
	}
}

func TestExporter_Export_SchemaPathInFrontmatter(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Schema:  &catalog.Schema{Type: "object"},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(
		filepath.Join(tmpDir, "services", "svc", "commands", "CreateOrder", "index.mdx"),
	)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "schemaPath: schemas/schema.json") {
		t.Errorf("message frontmatter missing schemaPath: %s", content)
	}
}

func TestExporter_Export_NoSchemaPathWhenNoSchema(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "NoSchema",
		Name:    "NoSchema",
		Version: "1.0.0",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(
		filepath.Join(tmpDir, "services", "svc", "commands", "NoSchema", "index.mdx"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "schemaPath") {
		t.Error("schemaPath should not appear when no schema provided")
	}
}

func TestExporter_Export_ServiceSendsReceives(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddEvent("order-svc", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "OrderCreated",
		Name:      "OrderCreated",
		Version:   "1.0.0",
		Direction: catalog.Sends,
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "PaymentProcessed",
		Name:      "PaymentProcessed",
		Version:   "2.0.0",
		Direction: catalog.Receives,
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	cattest.AssertContentContains(t, content, "service file",
		"id: order-svc",
		"sends:",
		"- OrderCreated/1.0.0",
		"receives:",
		"- PaymentProcessed/2.0.0",
	)
}

func TestExporter_Export_YAMLFrontmatter(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Service", Version: "2.0.0", Summary: "A test service",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "services", "svc", "index.mdx"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Error("MDX file should start with YAML frontmatter")
	}

	if !strings.Contains(content, "---\n\n# ") {
		t.Error("MDX file should have content after frontmatter with heading")
	}
}

func TestExporter_Export_CommandsAndQueriesInServiceFrontmatter(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
	})
	cattest.AddMessageSimple(t, reg, "order-svc", "GetOrder", "GetOrder", "1.0.0", "", catalog.QueryMessage, reg.AddQuery)

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	cattest.AssertContentContains(t, content, "service file",
		"commands:",
		"- CreateOrder/1.0.0",
		"queries:",
		"- GetOrder/1.0.0",
	)
}

func TestExporter_Export_LLMsTxt(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("MyCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Summary: "Create a new order",
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "OrderCreated",
		Name:      "OrderCreated",
		Version:   "1.0.0",
		Summary:   "Order was created",
		Direction: catalog.Sends,
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      "GetOrder",
		Name:    "GetOrder",
		Version: "1.0.0",
		Summary: "Get order by ID",
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "llms.txt"))
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	content := string(data)
	cattest.AssertContentContains(t, content, "llms.txt",
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
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "CreateOrder",
		Version: "1.0.0",
		Examples: []json.RawMessage{
			[]byte(`{"orderId":"abc-123","amount":42.5}`),
		},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	examplesPath := filepath.Join(
		tmpDir,
		"services",
		"svc",
		"commands",
		"CreateOrder",
		"examples.json",
	)

	data, err := os.ReadFile(examplesPath)
	if err != nil {
		t.Fatalf("read examples.json: %v", err)
	}

	content := string(data)
	cattest.AssertContentContains(t, content, "examples.json",
		"orderId",
		"42.5",
	)
}

func TestExporter_Export_PackageJSON(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("My Catalog", "2.0.0")
	cat := reg.Build()

	exp := NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	pkgPath := filepath.Join(tmpDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"name": "my-catalog"`) {
		t.Errorf("package.json missing name: %s", content)
	}

	if !strings.Contains(content, `"version": "2.0.0"`) {
		t.Errorf("package.json missing version: %s", content)
	}

	if !strings.Contains(content, `"private": true`) {
		t.Errorf("package.json missing private: %s", content)
	}

	if !strings.Contains(content, "@eventcatalog/core") {
		t.Errorf("package.json missing eventcatalog dependency: %s", content)
	}
}
