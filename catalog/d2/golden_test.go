package d2_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestGolden_D2Export(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	exp := d2.NewExporter("E-Commerce API", "1.0.0", d2.WithDescription("System overview"))
	got := exp.Export(cat)

	matchD2Golden(t, "diagram", got)
}

func TestGolden_D2WithOps(t *testing.T) {
	cat := cattest.BuildTestCatalogWithOps()
	exp := d2.NewExporter("REST API", "1.0.0")
	got := exp.Export(cat)

	matchD2Golden(t, "diagram-with-ops", got)
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

func matchD2Golden(t *testing.T, name, got string) {
	t.Helper()
	snaps.WithConfig(
		snaps.Dir(filepath.Join("..", "testdata", "golden")),
		snaps.Filename(name),
	).MatchSnapshot(t, normalizeD2(got))
}
