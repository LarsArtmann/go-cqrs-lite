package eventcatalog_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var update = flag.Bool("update", false, "update golden files")

func goldenDir() string {
	return filepath.Join("..", "testdata", "golden")
}

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

func readOrWriteGolden(t *testing.T, goldenPath, actualContent string) {
	t.Helper()

	if *update {
		err := os.WriteFile(goldenPath, []byte(actualContent), 0o644)
		if err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}

	if strings.TrimSpace(actualContent) != strings.TrimSpace(string(want)) {
		t.Errorf("golden file %s mismatch (run with -update to refresh)", goldenPath)
	}
}

func TestGolden_EventCatalog_ServiceMDX(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	svcContent, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "eventcatalog-service.mdx"), string(svcContent))
}

func TestGolden_EventCatalog_Config(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	cfgContent, err := os.ReadFile(filepath.Join(tmpDir, "eventcatalog.config.js"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "eventcatalog-config.js"), string(cfgContent))
}

func TestGolden_EventCatalog_LLMsTxt(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	llmsContent, err := os.ReadFile(filepath.Join(tmpDir, "llms.txt"))
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "llms.txt"), string(llmsContent))
}

func TestGolden_EventCatalog_PackageJSON(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := exportToTempDir(t, cat)

	pkgContent, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	readOrWriteGolden(t, filepath.Join(goldenDir(), "package.json"), string(pkgContent))
}
