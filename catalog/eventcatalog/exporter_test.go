package eventcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func newCommand(id string) catalog.Message {
	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      catalog.MessageID(id),
		Name:    catalog.Name(id),
		Version: "1.0.0",
	}

	return msg
}

func newEvent(id, name string, direction catalog.Direction) catalog.Message {
	return catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        catalog.MessageID(id),
		Name:      catalog.Name(name),
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

func assertFileExists(t *testing.T, tmpDir, msg string, parts ...string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(append([]string{tmpDir}, parts...)...)); os.IsNotExist(err) {
		t.Error(msg)
	}
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

	cmdSchema, err := cattest.StringSchema("orderId", "timestamp")
	if err != nil {
		t.Fatal(err)
	}

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
		Schema:  cmdSchema,
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
