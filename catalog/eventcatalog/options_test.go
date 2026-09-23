package eventcatalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

// TestExporter_DefaultWritesBootstrapFiles pins the default contract: local
// export directories stay directly buildable (npm install && eventcatalog
// build), so both bootstrap files must exist.
func TestExporter_DefaultWritesBootstrapFiles(t *testing.T) {
	tmpDir := exportToTempDir(t, cattest.BuildTestCatalog())

	for _, bootstrap := range []string{"eventcatalog.config.js", "package.json"} {
		if _, err := os.Stat(filepath.Join(tmpDir, bootstrap)); err != nil {
			t.Errorf("default export should write %s: %v", bootstrap, err)
		}
	}
}

// TestExporter_SkipBootstrapFilesOmitsOnlyBootstrap pins the option's
// contract: exactly the two bootstrap files disappear; every resource, the
// manifest, and the AI-consumption files are untouched.
func TestExporter_SkipBootstrapFilesOmitsOnlyBootstrap(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir, eventcatalog.WithSkipBootstrapFiles())
	if err := exp.Export(cat); err != nil {
		t.Fatalf("export: %v", err)
	}

	for _, bootstrap := range []string{"eventcatalog.config.js", "package.json"} {
		if _, err := os.Stat(filepath.Join(tmpDir, bootstrap)); !os.IsNotExist(err) {
			t.Errorf("skip-bootstrap export must not write %s (stat err: %v)", bootstrap, err)
		}
	}

	stillThere := []string{
		"services/order-svc/index.mdx",
		"events/OrderCreated/index.mdx",
		"catalog.index.json",
		"llms.txt",
		"schemas.txt",
	}
	for _, rel := range stillThere {
		if _, err := os.Stat(filepath.Join(tmpDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("skip-bootstrap export must still write %s: %v", rel, err)
		}
	}
}

// TestExporter_SkipBootstrapFilesLeavesContentIdentical proves the option
// never changes CONTENT — a hub merging a skip-bootstrap export gets exactly
// the same MDX bytes as a default export.
func TestExporter_SkipBootstrapFilesLeavesContentIdentical(t *testing.T) {
	cat := cattest.BuildTestCatalog()

	defaultDir := exportToTempDir(t, cat)
	skipDir := t.TempDir()

	exp := eventcatalog.NewExporter(skipDir, eventcatalog.WithSkipBootstrapFiles())
	if err := exp.Export(cat); err != nil {
		t.Fatalf("export: %v", err)
	}

	rel := filepath.Join("events", "OrderCreated", "index.mdx")

	defaultContent, err := os.ReadFile(filepath.Join(defaultDir, rel))
	if err != nil {
		t.Fatalf("read default export: %v", err)
	}

	skipContent, err := os.ReadFile(filepath.Join(skipDir, rel))
	if err != nil {
		t.Fatalf("read skip-bootstrap export: %v", err)
	}

	if string(defaultContent) != string(skipContent) {
		t.Errorf(
			"skip-bootstrap export changed content of %s:\ndefault:\n%s\nskip:\n%s",
			rel, defaultContent, skipContent,
		)
	}
}
