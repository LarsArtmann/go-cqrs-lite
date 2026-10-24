package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

type leakOrder struct {
	ID string `json:"id" doc:"Identifier"`
}

// leakCatalog carries a data store whose name is a hostile string — a URL
// that reads like a secret — plus a service that references it, so every
// reference surface (link, owner-style text, channel list, search index)
// would carry the name if a redaction leaked.
func leakCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("LeakCo", "1.0.0")

	const hostile = "https://internal.example.com/secret/quarterly-revenue"

	builder.AddService("leak-service", "Leak Service", "1.0.0", "Reads the secret store.",
		catalog.Command[leakOrder]("leak-command"),
	)

	builder.ConfigureService("leak-service",
		catalog.ServiceReadsFrom("secret-store"),
	)

	builder.AddDataStore(catalog.DataStore{
		ID:         "secret-store",
		Name:       catalog.Name(hostile),
		Version:    "1.0.0",
		Summary:    "Quarterly revenue, unreleased.",
		Technology: "postgres@16",
	})

	return builder.Build()
}

func renderLeak(t *testing.T, opts ...architecture.Option) string {
	t.Helper()

	base := []architecture.Option{
		architecture.WithTitle("LeakCo"),
		architecture.WithCommand("catalog-export -architecture"),
	}

	out, err := architecture.NewExporter(append(base, opts...)...).Export(leakCatalog())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	return string(out)
}

// TestExcludedResourceNameNeverAppears is the redaction leak test: once a
// resource is excluded, its name appears nowhere in the document — not as a
// link, not as plain text, not in the search index — in the default omit
// mode and in name-only mode alike.
func TestExcludedResourceNameNeverAppears(t *testing.T) {
	const hostile = "https://internal.example.com/secret/quarterly-revenue"

	for _, mode := range []struct {
		name string
		opt  architecture.Option
	}{
		{"omit", architecture.WithReferences(architecture.ReferencesOmit)},
		{"name-only", architecture.WithReferences(architecture.ReferencesNameOnly)},
		{"include", architecture.WithReferences(architecture.ReferencesInclude)},
	} {
		t.Run(mode.name, func(t *testing.T) {
			out := renderLeak(t,
				architecture.WithExclude(
					architecture.Selector{Kind: "container", ID: "secret-store"},
				),
				mode.opt,
			)

			if strings.Contains(out, hostile) {
				t.Error("the excluded store's name leaked into the export")
			}

			if strings.Contains(out, `id="data-store-secret-store"`) {
				t.Error("the excluded store has a section")
			}
		})
	}

	t.Run("without the exclude the name is searchable", func(t *testing.T) {
		out := renderLeak(t)

		if !strings.Contains(out, hostile) {
			t.Fatal("the fixture stopped carrying the hostile name — the test guards nothing")
		}
	})
}

// TestLoadExportValidatesReferencesMode pins the references-vocabulary gate
// (ADR-0160): a typo'd mode is a Rejection and "omit" is a valid spelling of
// the default.
func TestLoadExportValidatesReferencesMode(t *testing.T) {
	dir := t.TempDir()

	write := func(name, references string) string {
		t.Helper()

		path := filepath.Join(dir, name)

		content := "---\ntitle: Mode slice\ninclude:\n  - kind: domain\n    id: d\nreferences: " +
			references + "\n---\nbody\n"

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		return path
	}

	t.Run("typo is a rejection", func(t *testing.T) {
		_, err := architecture.LoadExport(write("typo.mdx", "name-onlyx"))
		if err == nil {
			t.Fatal("an unknown references mode must be rejected")
		}

		if !strings.Contains(err.Error(), "references") {
			t.Errorf("the rejection does not name the offending key: %v", err)
		}
	})

	t.Run("omit is a valid spelling of the default", func(t *testing.T) {
		spec, err := architecture.LoadExport(write("omit.mdx", "omit"))
		if err != nil {
			t.Fatalf("LoadExport: %v", err)
		}

		if spec.References != architecture.ReferencesOmit {
			t.Errorf("references: %q", spec.References)
		}
	})
}

// TestLoadExportCatalogTitleSentinel pins the "catalog" title sentinel: the
// spec sets no title option, so the exporter falls back to the catalog's own
// title — one spec file can travel across catalogs (ADR-0160).
func TestLoadExportCatalogTitleSentinel(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "sentinel.mdx")

	const content = "---\ntitle: catalog\ninclude:\n  - kind: domain\n    id: ordering\n---\nbody\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	spec, err := architecture.LoadExport(path)
	if err != nil {
		t.Fatalf("LoadExport: %v", err)
	}

	out := renderSelection(t, spec.Options()...)

	if !strings.Contains(out, "Acme · Architecture reference") {
		t.Error("the catalog sentinel did not fall back to the catalog's own title")
	}
}

// TestLoadExportParsesSpec pins the custom-export frontmatter contract:
// title and include are required, the body becomes the introduction, and
// x-* selectors are rejected pointing at ADR-0160.
func TestLoadExportParsesSpec(t *testing.T) {
	dir := t.TempDir()

	write := func(name, content string) string {
		t.Helper()

		path := filepath.Join(dir, name)

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		return path
	}

	valid := write("orders.mdx", `---
title: Orders slice
description: What ordering owns.
include:
  - kind: domain
    id: ordering
exclude:
  - kind: container
    id: audit-db
hide: [owners]
references: name-only
---

Everything about **orders**, redacted for the partner portal.
`)

	spec, err := architecture.LoadExport(valid)
	if err != nil {
		t.Fatalf("LoadExport: %v", err)
	}

	if spec.Title != "Orders slice" {
		t.Errorf("title: %q", spec.Title)
	}

	if len(spec.Include) != 1 || spec.Include[0].Kind != "domain" ||
		spec.Include[0].ID != "ordering" {
		t.Errorf("include selectors: %+v", spec.Include)
	}

	if len(spec.Exclude) != 1 || spec.Exclude[0].ID != "audit-db" {
		t.Errorf("exclude selectors: %+v", spec.Exclude)
	}

	if spec.References != architecture.ReferencesNameOnly {
		t.Errorf("references: %q", spec.References)
	}

	if len(spec.Hide) != 1 || spec.Hide[0] != architecture.HideOwners {
		t.Errorf("hide: %v", spec.Hide)
	}

	if !strings.Contains(spec.Introduction, "**orders**") {
		t.Errorf("introduction lost the body: %q", spec.Introduction)
	}

	t.Run("spec options render the introduction", func(t *testing.T) {
		out := renderSelection(t, spec.Options()...)

		if !strings.Contains(out, "partner portal") {
			t.Error("the introduction does not render in the overview")
		}

		if !strings.Contains(out, `id="domain-ordering"`) {
			t.Error("the spec's include selector did not keep the domain")
		}
	})

	for name, content := range map[string]string{
		"no fence":     "title: x\n",
		"unclosed":     "---\ntitle: x\n",
		"no title":     "---\ninclude:\n  - kind: domain\n    id: d\n---\nbody\n",
		"no include":   "---\ntitle: Only a title\n---\nbody\n",
		"custom prop ": "---\ntitle: T\ninclude:\n  - kind: x-secret\n    id: s\n---\nbody\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := write(strings.ReplaceAll(strings.TrimSpace(name), " ", "-")+".md", content)

			if _, err := architecture.LoadExport(path); err == nil {
				t.Errorf("%s must be rejected", name)
			}
		})
	}
}
