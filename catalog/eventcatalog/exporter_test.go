package eventcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

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
				"orderId": {Type: "string"},
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

	if !strings.Contains(content, "name: Order Service") {
		t.Errorf("service file missing name: %s", content)
	}

	if !strings.Contains(content, "# Order Service") {
		t.Errorf("service file missing heading: %s", content)
	}

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
	if !strings.Contains(content, "id: CreateOrder") {
		t.Errorf("command file missing id: %s", content)
	}

	if !strings.Contains(content, "owners:") {
		t.Errorf("command file missing owners: %s", content)
	}

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
	if !strings.Contains(content, "id: PaymentCompleted") {
		t.Errorf("event file missing id: %s", content)
	}

	if !strings.Contains(content, "# PaymentCompleted") {
		t.Errorf("event file missing heading: %s", content)
	}
}

func TestExporter_Export_Query(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "catalog-svc", Name: "Catalog Service", Version: "1.0.0",
	})
	reg.AddQuery("catalog-svc", catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      "GetProduct",
		Name:    "GetProduct",
		Version: "1.0.0",
	})

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

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddDomain(catalog.Domain{
		ID: "ordering", Name: "Ordering", Version: "1.0.0", Summary: "Order management domain",
		Services: []string{"order-svc"},
	})

	cat := reg.Build()

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
	if !strings.Contains(content, "id: ordering") {
		t.Errorf("domain file missing id: %s", content)
	}

	if !strings.Contains(content, "services:") {
		t.Errorf("domain file missing services list: %s", content)
	}

	if !strings.Contains(content, "- order-svc") {
		t.Errorf("domain file missing service reference: %s", content)
	}
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
	reg.AddCommand("svc-a", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CmdA", Name: "CmdA", Version: "1.0.0",
	})
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
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "NoSchema", Name: "NoSchema", Version: "1.0.0",
	})

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
	if !strings.Contains(content, "sends:") {
		t.Errorf("service frontmatter missing sends: %s", content)
	}

	if !strings.Contains(content, "- OrderCreated/1.0.0") {
		t.Errorf("service frontmatter missing sends entry: %s", content)
	}

	if !strings.Contains(content, "receives:") {
		t.Errorf("service frontmatter missing receives: %s", content)
	}

	if !strings.Contains(content, "- PaymentProcessed/2.0.0") {
		t.Errorf("service frontmatter missing receives entry: %s", content)
	}
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
