package eventcatalog_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

var update = flag.Bool("update", false, "update golden files")

func goldenDir() string {
	return filepath.Join("..", "testdata", "golden")
}

func buildTestCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("E-Commerce", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order", Version: "1.0.0",
		Summary: "Create a new order",
		Schema: &catalog.Schema{Type: "object", Properties: map[string]catalog.Property{
			"orderId": {Type: "string"},
		}},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created", Version: "1.0.0",
		Summary: "Order was created", Direction: catalog.Sends,
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order", Version: "1.0.0",
		Summary: "Get order by ID",
	})

	return reg.Build()
}

func exportToTempDir(t *testing.T, cat *catalog.Catalog) string {
	t.Helper()

	tmpDir := t.TempDir()
	exp := eventcatalog.NewExporter(tmpDir)
	if err := exp.Export(cat); err != nil {
		t.Fatalf("export: %v", err)
	}

	return tmpDir
}

func readOrWriteGolden(t *testing.T, goldenPath string, actualContent string) {
	t.Helper()

	if *update {
		if err := os.WriteFile(goldenPath, []byte(actualContent), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}

	if actualContent != string(want) {
		t.Errorf("golden file %s mismatch (run with -update to refresh)", goldenPath)
	}
}

func TestGolden_EventCatalog_ServiceMDX(t *testing.T) {
	cat := buildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	svcContent, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "eventcatalog-service.mdx"), string(svcContent))
}

func TestGolden_EventCatalog_Config(t *testing.T) {
	cat := buildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	cfgContent, err := os.ReadFile(filepath.Join(tmpDir, "eventcatalog.config.js"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "eventcatalog-config.js"), string(cfgContent))
}

func TestGolden_EventCatalog_LLMsTxt(t *testing.T) {
	cat := buildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	llmsContent, err := os.ReadFile(filepath.Join(tmpDir, "llms.txt"))
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "llms.txt"), string(llmsContent))
}

func TestGolden_EventCatalog_PackageJSON(t *testing.T) {
	cat := buildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	pkgContent, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "package.json"), string(pkgContent))
}
