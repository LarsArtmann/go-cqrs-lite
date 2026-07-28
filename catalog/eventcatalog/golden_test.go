package eventcatalog_test

import (
	"os"
	"path/filepath"
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
