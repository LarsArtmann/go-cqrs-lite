package adapters_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

func TestBuilder_ExportEventCatalog(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	builder := adapters.NewBuilder("E-Commerce", "1.0.0")
	builder.AddService(
		"order-svc", "Order Service", "1.0.0", "Manages orders",
		catalog.Command[createUserCmd]("order.create"),
	)

	err := builder.ExportEventCatalog(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	svcPath := filepath.Join(tmpDir, "services", "order-svc", "index.mdx")
	cattest.AssertFileExists(t, svcPath)

	cmdPath := filepath.Join(
		tmpDir,
		"services",
		"order-svc",
		"commands",
		"order.create",
		"index.mdx",
	)
	cattest.AssertFileExists(t, cmdPath)

	cfgPath := filepath.Join(tmpDir, "eventcatalog.config.js")
	cattest.AssertFileExists(t, cfgPath)
}

func TestBuilder_ExportAsyncAPI(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("E-Commerce", "1.0.0")
	builder.AddService(
		"order-svc", "Order Service", "1.0.0", "",
		catalog.Command[createUserCmd]("order.create"),
	)

	doc, err := builder.ExportAsyncAPI(
		"E-Commerce API", "1.0.0",
		asyncapi.WithServer("production", "kafka:9092", "kafka"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if doc.AsyncAPI != "3.0.0" {
		t.Errorf("asyncapi = %q, want 3.0.0", doc.AsyncAPI)
	}

	if len(doc.Channels) == 0 {
		t.Error("expected at least one channel")
	}

	srv, ok := doc.Servers["production"]
	if !ok {
		t.Fatal("missing production server")
	}

	if srv.Host != "kafka:9092" {
		t.Errorf("server host = %q", srv.Host)
	}
}

func TestBuilder_ExportD2(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"order-svc", "Order Service", "1.0.0", "Manages orders",
		catalog.Command[createUserCmd]("order.create"),
	)

	result := builder.ExportD2("Test API", "1.0.0")
	if result == "" {
		t.Error("ExportD2 returned empty string")
	}
}

func TestBuilder_ExportOpenAPI(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "",
		catalog.Command[createUserCmd]("user.create"),
	)

	doc := builder.ExportOpenAPI("Test Service", "1.0.0")
	if doc.OpenAPI != "3.0.3" {
		t.Errorf("openapi = %q, want 3.0.3", doc.OpenAPI)
	}

	if doc.Info.Title != "Test Service" {
		t.Errorf("title = %q, want Test Service", doc.Info.Title)
	}
}

func TestBuilder_AddCommand(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("svc", "Svc", "1.0.0", "")
	builder.AddCommand("svc", catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        "test-cmd",
		Name:      "TestCmd",
		Version:   "1.0.0",
		Direction: catalog.Receives,
	})

	cat := builder.Build()
	if len(cat.Services[0].Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(cat.Services[0].Commands))
	}

	if cat.Services[0].Commands[0].ID != "test-cmd" {
		t.Errorf("ID = %q, want test-cmd", cat.Services[0].Commands[0].ID)
	}
}

func TestBuilder_AddEvent(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("svc", "Svc", "1.0.0", "")
	builder.AddEvent("svc", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "test-evt",
		Name:      "TestEvt",
		Version:   "1.0.0",
		Direction: catalog.Sends,
	})

	cat := builder.Build()
	if len(cat.Services[0].Events) != 1 {
		t.Fatalf("events = %d, want 1", len(cat.Services[0].Events))
	}

	if cat.Services[0].Events[0].ID != "test-evt" {
		t.Errorf("ID = %q, want test-evt", cat.Services[0].Events[0].ID)
	}
}

func TestBuilder_AddQuery(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("svc", "Svc", "1.0.0", "")
	builder.AddQuery("svc", catalog.Message{
		Kind:      catalog.QueryMessage,
		ID:        "test-qry",
		Name:      "TestQry",
		Version:   "1.0.0",
		Direction: catalog.Receives,
	})

	cat := builder.Build()
	if len(cat.Services[0].Queries) != 1 {
		t.Fatalf("queries = %d, want 1", len(cat.Services[0].Queries))
	}

	if cat.Services[0].Queries[0].ID != "test-qry" {
		t.Errorf("ID = %q, want test-qry", cat.Services[0].Queries[0].ID)
	}
}

func TestJSONToYAML(t *testing.T) {
	t.Parallel()

	obj := map[string]any{"type": "object"}
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	yamlBytes, err := adapters.JSONToYAML(jsonBytes)
	if err != nil {
		t.Fatalf("JSONToYAML: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Fatal("yaml output is empty")
	}
}

func TestJSONToYAML_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := adapters.JSONToYAML([]byte("{invalid"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
