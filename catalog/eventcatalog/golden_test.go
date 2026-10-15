package eventcatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func exportToTempDir(t *testing.T, cat *catalog.Catalog) string {
	t.Helper()

	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir)

	err := exp.Export(cat)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	return tmpDir
}

func matchCatalogGolden(t *testing.T, name, got string) {
	t.Helper()
	snaps.WithConfig(
		snaps.Dir(filepath.Join("..", "testdata", "golden")),
		snaps.Filename(name),
	).MatchSnapshot(t, got)
}

func TestGolden_EventCatalog_ServiceMDX(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	svcContent, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}

	matchCatalogGolden(t, "eventcatalog-service", string(svcContent))
}

func TestGolden_EventCatalog_Config(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	cfgContent, err := os.ReadFile(filepath.Join(tmpDir, "eventcatalog.config.js"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	matchCatalogGolden(t, "eventcatalog-config", string(cfgContent))
}

func TestGolden_EventCatalog_LLMsTxt(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	llmsContent, err := os.ReadFile(filepath.Join(tmpDir, "llms.txt"))
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	matchCatalogGolden(t, "llms", string(llmsContent))
}

func TestGolden_EventCatalog_PackageJSON(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	pkgContent, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	matchCatalogGolden(t, "package", string(pkgContent))
}

func TestGolden_EventCatalog_IndexManifest(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	manifestContent, err := os.ReadFile(filepath.Join(tmpDir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read catalog.index.json: %v", err)
	}

	matchCatalogGolden(t, "catalog-index-manifest", string(manifestContent))
}

func TestGolden_EventCatalog_ADRMDX(t *testing.T) {
	reg := cattest.NewTestRegistry()
	reg.AddADR(catalog.ADR{
		ID:      "sqlite-single-writer",
		Name:    "SQLite single-writer event store",
		Version: "1.0.0",
		Status:  catalog.ADRAccepted,
		Date:    "2026-10-15",
		Summary: "One writer per database file keeps the sync window simple",
		Owners:  []string{"lars"},
		DecisionMakers: []string{
			"lars",
		},
		AppliesTo: []catalog.ADRResourceRef{
			{Type: catalog.ADRAppliesToService, ID: "bank-sync", Version: "1.0.0"},
			{Type: catalog.ADRAppliesToEvent, ID: "BalanceUpdated"},
		},
		Supersedes: []catalog.ADRRef{{ID: "sqlite-async-replication", Version: "0.9.0"}},
		Amends:     []catalog.ADRRef{{ID: "event-store-baseline"}},
		AmendedBy:  []catalog.ADRRef{{ID: "snapshot-every-100"}},
		Related:    []catalog.ADRRef{{ID: "wise-qonto-providers"}},
		Badges: []catalog.Badge{
			{Content: "Accepted", BackgroundColor: "green", TextColor: "white"},
		},
		Body: "## Context\n\nOne sync process writes balances.\n\n" +
			"## Decision\n\nSingle writer, no row locks.\n\n" +
			"## Consequences\n\nNo lock contention.",
	})

	tmpDir := exportToTempDir(t, reg.Build())

	adrContent, err := os.ReadFile(
		filepath.Join(tmpDir, "adrs", "sqlite-single-writer", "index.mdx"),
	)
	if err != nil {
		t.Fatalf("read adr file: %v", err)
	}

	matchCatalogGolden(t, "eventcatalog-adr", string(adrContent))
}

// Fixture types exercising the schema flattener: Payment embeds
// PaymentMethod (multi-level flattening), PlaceOrderPayload embeds two
// structs plus a plain field. The golden pins the FLATTENED output —
// embedded type names must never leak into the exported properties.
type auditFields struct {
	CorrelationID string `json:"correlationId"`
	Actor         string `json:"actor"`
}

type paymentMethod struct {
	Kind string `json:"kind"`
}

type payment struct {
	paymentMethod

	Reference string `json:"reference"`
}

type PlaceOrderPayload struct {
	auditFields
	payment

	OrderID string `json:"orderId"`
}

func TestGolden_EventCatalog_FlattenedSchema(t *testing.T) {
	cat := cattest.NewTestRegistry(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})
	cat.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "PlaceOrder",
		Name:    "Place Order",
		Version: "1.0.0",
		Summary: "Place a new order",
		Schema:  catalog.SchemaFromType[PlaceOrderPayload](),
	})

	tmpDir := exportToTempDir(t, cat.Build())

	schemaContent, err := os.ReadFile(
		filepath.Join(tmpDir, "commands", "PlaceOrder", "schemas", "schema.json"),
	)
	if err != nil {
		t.Fatalf("read flattened schema: %v", err)
	}

	for _, leaked := range []string{"auditFields", "paymentMethod", "\"payment\""} {
		if strings.Contains(string(schemaContent), leaked) {
			t.Errorf(
				"embedded type name %q leaked into flattened schema: %s",
				leaked,
				schemaContent,
			)
		}
	}

	matchCatalogGolden(t, "eventcatalog-flattened-schema", string(schemaContent))
}
