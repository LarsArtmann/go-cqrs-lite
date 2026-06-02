package asyncapi_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var update = flag.Bool("update", false, "update golden files")

func goldenDir() string {
	return filepath.Join("..", "testdata", "golden")
}

func TestGolden_AsyncAPIJSON(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	exp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "asyncapi.json")

	if *update {
		err := os.WriteFile(goldenPath, append(got, '\n'), 0o644)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("AsyncAPI JSON mismatch (run with -update to refresh golden files)")
	}
}

func TestGolden_AsyncAPIYAML(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	exp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "asyncapi.yaml")

	if *update {
		err := os.WriteFile(goldenPath, append(got, '\n'), 0o644)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("AsyncAPI YAML mismatch (run with -update to refresh golden files)")
	}
}
