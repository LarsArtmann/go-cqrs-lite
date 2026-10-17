package architecture_test

import (
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

func TestZZDump(t *testing.T) {
	out, err := architecture.NewExporter(
		architecture.WithTitle("Acme"),
		architecture.WithDescription("Acme's architecture."),
		architecture.WithGeneratedAt(excludeFrozenAt()),
		architecture.WithCommand("catalog-export -architecture"),
	).Export(fixtureCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/archref.html", out, 0o600); err != nil {
		t.Fatal(err)
	}
}
