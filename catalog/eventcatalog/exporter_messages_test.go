package eventcatalog

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestExporter_Export_Event(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(catalog.Service{
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

	reg := cattest.NewTestRegistry()
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

	reg := cattest.NewTestRegistry()
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
		"id: order-svc",
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

	reg := cattest.NewTestRegistry(
		catalog.Service{ID: "svc-a", Name: "Service A", Version: "1.0.0"},
	)
	reg.AddService(catalog.Service{ID: "svc-b", Name: "Service B", Version: "1.0.0"})
	reg.AddCommand("svc-a", newCommand("CmdA"))
	reg.AddEvent("svc-b", catalog.Message{
		Kind: catalog.EventMessage, ID: "EvtB", Name: "EvtB", Version: "1.0.0",
	})

	tmpDir := exportCatalog(t, reg)

	assertFileExists(t, tmpDir, "svc-a directory not created", "services", "svc-a", "index.mdx")
	assertFileExists(t, tmpDir, "svc-b directory not created", "services", "svc-b", "index.mdx")
	assertFileExists(
		t,
		tmpDir,
		"CmdA command file not created",
		"services",
		"svc-a",
		"commands",
		"CmdA",
		"index.mdx",
	)
	assertFileExists(
		t,
		tmpDir,
		"EvtB event file not created",
		"services",
		"svc-b",
		"events",
		"EvtB",
		"index.mdx",
	)
}
