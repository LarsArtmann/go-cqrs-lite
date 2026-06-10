package openapi_test

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/openapi"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var update = flag.Bool("update", false, "update golden files")

func TestGolden_OpenAPIJSON(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	exp := openapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	cattest.AssertGolden(t, filepath.Join(cattest.GoldenDir(), "openapi.json"),
		got, *update, "OpenAPI JSON mismatch (run with -update to refresh golden files)")
}
