package asyncapi_test

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var update = flag.Bool("update", false, "update golden files")

func TestGolden_AsyncAPIJSON(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	exp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	cattest.AssertGolden(t, filepath.Join(cattest.GoldenDir(), "asyncapi.json"),
		got, *update, "AsyncAPI JSON mismatch (run with -update to refresh golden files)")
}

func TestGolden_AsyncAPIYAML(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	exp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	cattest.AssertGolden(t, filepath.Join(cattest.GoldenDir(), "asyncapi.yaml"),
		got, *update, "AsyncAPI YAML mismatch (run with -update to refresh golden files)")
}

func TestGolden_AsyncAPIWithOps(t *testing.T) {
	cat := cattest.BuildTestCatalogWithOps()
	exp := asyncapi.NewExporter("REST API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	cattest.AssertGolden(
		t,
		filepath.Join(cattest.GoldenDir(), "asyncapi-with-ops.json"),
		got,
		*update,
		"AsyncAPI JSON (with ops) mismatch (run with -update to refresh golden files)",
	)
}
