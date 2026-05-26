package d2_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var d2Update = flag.Bool("update-d2", false, "update d2 golden files")

func TestGolden_D2Export(t *testing.T) {
	t.Parallel()

	cat := cattest.BuildTestCatalog()
	exp := d2.NewExporter("E-Commerce API", "1.0.0", d2.WithDescription("System overview"))
	got := exp.Export(cat)

	goldenPath := filepath.Join("..", "testdata", "golden", "diagram.d2")

	if *d2Update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if strings.TrimSpace(got) != strings.TrimSpace(string(want)) {
		t.Errorf("D2 diagram mismatch (run with -update-d2 to refresh golden files)")
	}
}
