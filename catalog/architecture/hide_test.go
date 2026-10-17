package architecture_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

type hideOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"`
}

// hideCatalog builds a catalog whose service carries every hideable surface:
// owners, a repository, specifications, a message example payload and a
// documentation page.
func hideCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("HideCo", "1.0.0")

	builder.AddService("hide-service", "Hide Service", "1.0.0", "Owns every hideable block.",
		catalog.Command[hideOrder]("hide-command",
			catalog.WithExample(map[string]any{"order_id": "o-1"}),
		),
	)

	builder.ConfigureService("hide-service",
		catalog.ServiceOwners("hide-team"),
		catalog.ServiceRepository("go", "https://github.com/hideco/hide-service"),
		catalog.ServiceSpecifications(catalog.Specification{
			Type: "openapi",
			Path: "specs/hide.yaml",
			Name: catalog.Name("Hide API"),
		}),
	)

	builder.AddTeam(catalog.Team{ID: "hide-team", Name: "Hide Team", Summary: "Owns hiding."})
	builder.AddCustomDoc(catalog.CustomDoc{
		ID:      "hide-doc",
		Title:   "Hide handbook",
		Summary: "A documentation page.",
		Content: "Read the [Hide Service](#service-hide-service).\n",
	})

	return builder.Build()
}

func renderHide(t *testing.T, opts ...architecture.Option) string {
	t.Helper()

	base := []architecture.Option{
		architecture.WithTitle("HideCo"),
		architecture.WithCommand("catalog-export -architecture"),
	}

	out, err := architecture.NewExporter(append(base, opts...)...).Export(hideCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	return string(out)
}

// TestHideDefaults pins that without WithHide every hideable surface renders.
func TestHideDefaults(t *testing.T) {
	out := renderHide(t)

	for _, want := range []string{
		"Owned by",
		`href="#team-hide-team"`,
		"Source code",
		"Specifications",
		"Example payload",
		`id="doc-hide-doc"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("default export is missing %q", want)
		}
	}
}

// TestHideEmptyShowsAll pins that WithHide() with no sections is the default
// export, byte for byte.
func TestHideEmptyShowsAll(t *testing.T) {
	defaultOut := renderHide(t)
	emptyHide := renderHide(t, architecture.WithHide())

	if defaultOut != emptyHide {
		t.Error("WithHide() with no sections differs from the default export")
	}
}

// TestHideSections pins that each hideable surface disappears under its
// constant and that the others survive.
func TestHideSections(t *testing.T) {
	t.Run("owners", func(t *testing.T) {
		out := renderHide(t, architecture.WithHide(architecture.HideOwners))

		for _, gone := range []string{"Owned by", `href="#team-hide-team"`} {
			if strings.Contains(out, gone) {
				t.Errorf("owners still render: %q", gone)
			}
		}

		if !strings.Contains(out, "Source code") {
			t.Error("hiding owners took the repository block with it")
		}
	})

	t.Run("repository", func(t *testing.T) {
		out := renderHide(t, architecture.WithHide(architecture.HideRepository))

		if strings.Contains(out, "Source code") {
			t.Error("repository link still renders")
		}

		if !strings.Contains(out, "Owned by") {
			t.Error("hiding the repository took the owners meta with it")
		}
	})

	t.Run("specifications", func(t *testing.T) {
		out := renderHide(t, architecture.WithHide(architecture.HideSpecifications))

		if strings.Contains(out, "Specifications") {
			t.Error("specifications block still renders")
		}
	})

	t.Run("examples", func(t *testing.T) {
		out := renderHide(t, architecture.WithHide(architecture.HideExamples))

		if strings.Contains(out, "Example payload") {
			t.Error("example payload still renders")
		}
	})

	t.Run("docs", func(t *testing.T) {
		out := renderHide(t, architecture.WithHide(architecture.HideDocs))

		if strings.Contains(out, `id="doc-hide-doc"`) {
			t.Error("documentation page still renders")
		}

		if !strings.Contains(out, `id="service-hide-service"`) {
			t.Error("hiding docs took the service section with it")
		}
	})
}
