package eventcatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

// TestIntegration_FullEventCatalogExport validates that a catalog with ALL
// resource types exports to a complete EventCatalog directory structure
// that would be accepted by the EventCatalog CLI.
func TestIntegration_FullEventCatalogExport(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("E-Commerce", "2.0.0")

	// Service with messages
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0",
		Summary: "Manages orders",
		Events: []catalog.Message{
			{
				Kind:      catalog.EventMessage,
				ID:        "OrderCreated",
				Name:      "OrderCreated",
				Version:   "1.0.0",
				Direction: catalog.Sends,
			},
		},
		Commands: []catalog.Message{
			{
				Kind:    catalog.CommandMessage,
				ID:      "CreateOrder",
				Name:    "CreateOrder",
				Version: "1.0.0",
			},
		},
	})

	// Domain with ubiquitous language
	reg.AddDomain(catalog.Domain{
		ID: "orders", Name: "Orders", Version: "1.0.0",
		Services: []catalog.ServiceID{"order-svc"},
		UbiquitousLanguage: []catalog.UbiquitousLanguageTerm{
			{Name: "Order", Description: "A purchase request"},
		},
	})

	// Entity with properties
	reg.AddEntity(catalog.Entity{
		ID: "order", Name: "Order", Version: "1.0.0",
		AggregateRoot: true, Identifier: "orderId",
		Properties: []catalog.EntityProperty{
			{Name: "orderId", Type: "string", Required: true},
		},
	})

	// Data store
	reg.AddDataStore(catalog.DataStore{
		ID: "order-db", Name: "Order DB", Version: "1.0.0",
		ContainerType: "database", Technology: "PostgreSQL",
	})

	// Data product
	reg.AddDataProduct(catalog.DataProduct{
		ID: "order-metrics", Name: "Order Metrics", Version: "1.0.0",
		Inputs: []catalog.Ref{{ID: "OrderCreated"}},
	})

	// Agent
	reg.AddAgent(catalog.Agent{
		ID: "order-bot", Name: "Order Bot", Version: "1.0.0",
		Receives: []catalog.Ref{{ID: "OrderCreated"}},
	})

	// Channel
	reg.AddChannel(catalog.Channel{
		ID: "order-channel", Name: "Order Channel", Version: "1.0.0",
		Address: "orders.events", Protocols: []catalog.Protocol{"kafka"},
	})

	// Flow
	reg.AddFlow(catalog.Flow{
		ID: "checkout", Name: "Checkout Flow", Version: "1.0.0",
		Steps: []catalog.FlowStep{
			{ID: "s1", Title: "Customer orders", Service: &catalog.FlowStepRef{ID: "order-svc"}},
		},
	})

	// Team
	reg.AddTeam(catalog.Team{ID: "order-team", Name: "Order Team"})

	// User
	reg.AddUser(catalog.User{ID: "alice", Name: "Alice"})

	// Custom doc
	reg.AddCustomDoc(catalog.CustomDoc{
		ID:      "architecture",
		Title:   "Architecture",
		Content: "## Overview\nMicroservices architecture.",
	})

	cat := reg.Build()

	tmpDir := t.TempDir()
	exp := eventcatalog.NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify complete directory structure
	expectedFiles := []string{
		"services/order-svc/index.mdx",
		"services/order-svc/commands/CreateOrder/index.mdx",
		"services/order-svc/events/OrderCreated/index.mdx",
		"domains/orders/index.mdx",
		"entities/order/index.mdx",
		"data/order-db/index.mdx",
		"data-products/order-metrics/index.mdx",
		"agents/order-bot/index.mdx",
		"channels/order-channel/index.mdx",
		"flows/checkout/index.mdx",
		"teams/order-team.mdx",
		"users/alice.mdx",
		"docs/architecture/index.mdx",
		"eventcatalog.config.js",
		"llms.txt",
	}

	for _, relPath := range expectedFiles {
		fullPath := filepath.Join(tmpDir, relPath)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("missing expected file: %s (%v)", relPath, err)
		}
	}

	// Verify content has valid YAML frontmatter in all MDX files
	mdxFiles := []string{
		"services/order-svc/index.mdx",
		"domains/orders/index.mdx",
		"entities/order/index.mdx",
		"agents/order-bot/index.mdx",
		"data-products/order-metrics/index.mdx",
	}

	for _, relPath := range mdxFiles {
		fullPath := filepath.Join(tmpDir, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("read %s: %v", relPath, err)
		}

		content := string(data)
		if !strings.HasPrefix(content, "---\n") {
			t.Errorf("%s: expected MDX to start with frontmatter delimiter", relPath)
		}

		if !strings.Contains(content, "\n---\n") {
			t.Errorf("%s: expected closing frontmatter delimiter", relPath)
		}
	}
}

// TestIntegration_RestOperationsExport verifies that messages with explicit
// operations and typed responses export correct frontmatter and schema files.
func TestIntegration_RestOperationsExport(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("REST API", "1.0.0")

	reg.AddService(catalog.Service{
		ID: "user-svc", Name: "User Service", Version: "1.0.0",
		Summary: "User management REST API",
		Commands: []catalog.Message{
			{
				Kind: catalog.CommandMessage,
				ID:   "CreateUser", Name: "CreateUser", Version: "1.0.0",
				Operation: &catalog.Operation{
					Method: "POST", Path: "/api/users", StatusCodes: []string{"201", "400"},
				},
				Responses: []catalog.ResponseSpec{
					{
						StatusCode: "201", Description: "User created",
						Schema: &catalog.Schema{Type: catalog.TypeObject},
					},
				},
			},
		},
		Queries: []catalog.Message{
			{
				Kind: catalog.QueryMessage,
				ID:   "GetUser", Name: "GetUser", Version: "1.0.0",
				Operation: &catalog.Operation{Method: "GET", Path: "/api/users/{id}"},
			},
		},
	})

	cat := reg.Build()

	tmpDir := t.TempDir()
	exp := eventcatalog.NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify command MDX has operation and response frontmatter
	cmdPath := filepath.Join(tmpDir, "services/user-svc/commands/CreateUser/index.mdx")
	cmdData, err := os.ReadFile(cmdPath)
	if err != nil {
		t.Fatalf("read command MDX: %v", err)
	}

	cmdContent := string(cmdData)
	if !strings.Contains(cmdContent, "method: POST") {
		t.Error("command MDX missing operation method")
	}

	if !strings.Contains(cmdContent, "path: /api/users") {
		t.Error("command MDX missing operation path")
	}

	if !strings.Contains(cmdContent, "responses:") {
		t.Error("command MDX missing responses frontmatter")
	}

	if !strings.Contains(cmdContent, "statusCode: \"201\"") {
		t.Error("command MDX missing response status code")
	}

	// Verify query MDX has GET operation
	qryPath := filepath.Join(tmpDir, "services/user-svc/queries/GetUser/index.mdx")
	qryData, err := os.ReadFile(qryPath)
	if err != nil {
		t.Fatalf("read query MDX: %v", err)
	}

	qryContent := string(qryData)
	if !strings.Contains(qryContent, "method: GET") {
		t.Error("query MDX missing operation method")
	}

	if !strings.Contains(qryContent, "path: /api/users/{id}") {
		t.Error("query MDX missing operation path")
	}
}
