package eventcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/internal/cattest"
)

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
