package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

type composeOrder struct {
	ID string `json:"id" doc:"Identifier"`
}

// composeCatalog builds the composition fixture: a service with owners that
// reads from a store whose owners differ, so hide and the references pull
// have to compose across two sections with two distinct owner lists.
func composeCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("ComposeCo", "1.0.0")

	builder.AddService("compose-service", "Compose Service", "1.0.0", "Owns nothing it reads.",
		catalog.Command[composeOrder]("compose-command"),
	)

	builder.ConfigureService("compose-service",
		catalog.ServiceOwners("compose-team"),
		catalog.ServiceReadsFrom("partner-store"),
	)

	builder.AddDataStore(catalog.DataStore{
		ID:      "partner-store",
		Name:    "Partner Store",
		Version: "1.0.0",
		Summary: "Pulled in by reference.",
		Owners:  []string{"partner-team"},
	})

	builder.AddTeam(
		catalog.Team{ID: "compose-team", Name: "Compose Team", Summary: "Owns the service."},
	)
	builder.AddTeam(
		catalog.Team{ID: "partner-team", Name: "Partner Team", Summary: "Owns the store."},
	)

	return builder.Build()
}

// TestHideAndReferenceIncludeCompose pins the composition of the two
// redaction features (ADR-0160): a store pulled in by ReferencesInclude is a
// full section, so WithHide(HideOwners) must suppress its owners block the
// same way it suppresses the included service's.
func TestHideAndReferenceIncludeCompose(t *testing.T) {
	base := []architecture.Option{
		architecture.WithTitle("ComposeCo"),
		architecture.WithCommand("catalog-export -architecture"),
		architecture.WithInclude(architecture.Selector{Kind: "service", ID: "compose-service"}),
		architecture.WithReferences(architecture.ReferencesInclude),
	}

	t.Run("hide suppresses owners in pulled-in sections too", func(t *testing.T) {
		opts := append(base, architecture.WithHide(architecture.HideOwners))

		out, err := architecture.NewExporter(opts...).Export(composeCatalog())
		if err != nil {
			t.Fatalf("Export: %v", err)
		}

		if !strings.Contains(string(out), `id="data-store-partner-store"`) {
			t.Error("the include pull did not bring the store in")
		}

		if strings.Contains(string(out), "Owned by") {
			t.Error("owners meta rendered despite WithHide(HideOwners)")
		}

		if strings.Contains(string(out), "Compose Team") ||
			strings.Contains(string(out), "Partner Team") {
			t.Error("an owner name rendered despite WithHide(HideOwners)")
		}
	})

	t.Run("without hide the owners meta is the control", func(t *testing.T) {
		out, err := architecture.NewExporter(base...).Export(composeCatalog())
		if err != nil {
			t.Fatalf("Export: %v", err)
		}

		if !strings.Contains(string(out), "Owned by") {
			t.Fatal("the fixture stopped rendering owners meta — the test guards nothing")
		}
	})
}

// TestSpecAndFlagPrecedence pins how a spec file and CLI flags compose
// (ADR-0160): the spec's options apply first, the flags extend them — a flag
// exclude adds to the spec's, and a flag title overrides the spec's.
func TestSpecAndFlagPrecedence(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "slice.mdx")

	const spec = `---
title: Spec Slice
include:
  - kind: service
    id: order-service
exclude:
  - kind: container
    id: order-db
references: name-only
---

The partner slice.
`

	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	loaded, err := architecture.LoadExport(path)
	if err != nil {
		t.Fatalf("LoadExport: %v", err)
	}

	flags := []architecture.Option{
		architecture.WithTitle("Flag Title"),
		architecture.WithExclude(architecture.Selector{Kind: "team", ID: "ordering"}),
	}

	out, err := architecture.NewExporter(append(loaded.Options(), flags...)...).
		Export(fixtureCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if !strings.Contains(string(out), "Flag Title") || strings.Contains(string(out), "Spec Slice") {
		t.Error("the flag title did not override the spec title")
	}

	if !strings.Contains(string(out), `id="service-order-service"`) {
		t.Error("the spec's include did not survive the flag phase")
	}

	if strings.Contains(string(out), "Order Database") {
		t.Error("the spec's exclude stopped holding under name-only mode")
	}

	if strings.Contains(string(out), "Ordering Team") {
		t.Error("the flag's exclude did not extend the spec's (name-only rendered the banned team)")
	}

	if !strings.Contains(string(out), `<span class="rl rl-plain">Order</span>`) {
		t.Error("the spec's name-only mode did not ride through the composition")
	}
}

// TestComposedRedactionNeverLeaks is the composed-path leak probe (ADR-0160):
// on the leak fixture, with the store excluded, no composition of include,
// references mode and hide options may resurrect the hostile name — as text,
// as a link, or as a section. Assertions are text-presence based so they do
// not break on markup shape.
func TestComposedRedactionNeverLeaks(t *testing.T) {
	const hostile = "https://internal.example.com/secret/quarterly-revenue"

	storeGone := func(t *testing.T, out string) {
		t.Helper()

		if strings.Contains(out, hostile) {
			t.Error("the excluded store's name leaked into the export")
		}

		if strings.Contains(out, `href="#data-store-secret-store"`) {
			t.Error("the excluded store is linked")
		}
	}

	t.Run("exclude + name-only over an included service", func(t *testing.T) {
		out := renderLeak(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "leak-service"}),
			architecture.WithExclude(architecture.Selector{Kind: "container", ID: "secret-store"}),
			architecture.WithReferences(architecture.ReferencesNameOnly),
		)

		storeGone(t, out)

		if !strings.Contains(out, "Leak Service") {
			t.Error("the included service itself is missing")
		}
	})

	t.Run("exclude + include-pull", func(t *testing.T) {
		out := renderLeak(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "leak-service"}),
			architecture.WithExclude(architecture.Selector{Kind: "container", ID: "secret-store"}),
			architecture.WithReferences(architecture.ReferencesInclude),
		)

		storeGone(t, out)

		if strings.Contains(out, `id="data-store-secret-store"`) {
			t.Error("the excluded store got a section via the pull")
		}

		if !strings.Contains(out, `id="command-leak-command"`) {
			t.Error("the service's message closure is missing")
		}
	})

	t.Run("hide composes on top", func(t *testing.T) {
		out := renderLeak(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "leak-service"}),
			architecture.WithExclude(architecture.Selector{Kind: "container", ID: "secret-store"}),
			architecture.WithReferences(architecture.ReferencesNameOnly),
			architecture.WithHide(architecture.HideOwners, architecture.HideExamples),
		)

		storeGone(t, out)

		if !strings.Contains(out, "Leak Service") {
			t.Error("the included service itself is missing")
		}
	})

	t.Run("control: without the exclude the name rides name-only", func(t *testing.T) {
		out := renderLeak(t,
			architecture.WithInclude(architecture.Selector{Kind: "service", ID: "leak-service"}),
			architecture.WithReferences(architecture.ReferencesNameOnly),
		)

		if !strings.Contains(out, hostile) {
			t.Fatal("the fixture stopped carrying the leak path — the test guards nothing")
		}
	})
}
