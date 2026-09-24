package eventcatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

// TestExporter_OwnersEmittedOnEveryOwnableKind pins the data-mesh ownership
// contract: `owners` frontmatter is emitted for EVERY ownable resource kind
// (services were always covered; messages, domains, flows, channels, data
// stores/entities/agents are the ones the @eventcatalog/linter
// best-practices/owner-required rule checks). A regression here re-breaks
// federation-hub ownership linting.
func TestExporter_OwnersEmittedOnEveryOwnableKind(t *testing.T) {
	const team = "order-team"

	reg := cattest.NewTestRegistry(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "orders",
		Owners: []string{team},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderPlaced", Name: "Order Placed",
		Version: "1.0.0", Summary: "placed", Direction: catalog.Sends,
		Owners: []string{team},
	})
	reg.AddChannel(catalog.Channel{
		ID: "orders", Name: "Orders", Version: "1.0.0", Summary: "order messages",
		Owners: []string{team},
	})
	reg.AddDomain(catalog.Domain{
		ID: "ordering", Name: "Ordering", Version: "1.0.0", Summary: "ordering",
		Owners: []string{team},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders DB", Version: "1.0.0",
		ContainerType: "database", Owners: []string{team},
	})
	reg.AddEntity(catalog.Entity{
		ID: "Order", Name: "Order", Version: "1.0.0", Owners: []string{team},
	})
	reg.AddFlow(catalog.Flow{
		ID: "checkout", Name: "Checkout", Version: "1.0.0",
		Owners: []string{team},
	})
	reg.AddAgent(catalog.Agent{
		ID: "order-bot", Name: "Order Bot", Version: "1.0.0",
		Owners: []string{team},
	})
	reg.AddDataProduct(catalog.DataProduct{
		ID: "order-analytics", Name: "Order Analytics", Version: "1.0.0",
		Owners: []string{team},
	})

	tmpDir := exportToTempDir(t, reg.Build())

	resources := map[string]string{
		"service":      filepath.Join("services", "order-svc", "index.mdx"),
		"event":        filepath.Join("events", "OrderPlaced", "index.mdx"),
		"channel":      filepath.Join("channels", "orders", "index.mdx"),
		"domain":       filepath.Join("domains", "ordering", "index.mdx"),
		"data store":   filepath.Join("containers", "orders-db", "index.mdx"),
		"entity":       filepath.Join("entities", "Order", "index.mdx"),
		"flow":         filepath.Join("flows", "checkout", "index.mdx"),
		"agent":        filepath.Join("agents", "order-bot", "index.mdx"),
		"data product": filepath.Join("data-products", "order-analytics", "index.mdx"),
	}

	for kind, rel := range resources {
		data, err := os.ReadFile(filepath.Join(tmpDir, rel))
		if err != nil {
			t.Errorf("read %s export: %v", kind, err)

			continue
		}

		if !strings.Contains(string(data), "owners:\n    - "+team) {
			t.Errorf("%s frontmatter must emit owners:\n%s", kind, data)
		}
	}
}
