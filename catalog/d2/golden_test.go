package d2_test

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/internal/cattest"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var d2Update = flag.Bool("update-d2", false, "update d2 golden files")

func TestGolden_D2Export(t *testing.T) {
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

	gotNorm := normalizeD2(got)
	wantNorm := normalizeD2(string(want))
	if gotNorm != wantNorm {
		t.Errorf(
			"D2 diagram mismatch (run with -update-d2 to refresh golden files)\ngot len=%d, want len=%d",
			len(gotNorm),
			len(wantNorm),
		)
	}
}

var blankLineRe = regexp.MustCompile(`\n{2,}`)

func normalizeD2(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		cleaned = append(cleaned, strings.TrimRight(line, " \t"))
	}
	s = strings.Join(cleaned, "\n")
	s = blankLineRe.ReplaceAllString(s, "\n")
	return s
}
