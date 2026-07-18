// Command go-cqrs-lite-catalog generates documentation files (OpenAPI, AsyncAPI,
// D2, llms.txt) from a catalog definition.
//
// Usage:
//
//	go-cqrs-lite-catalog [format] [output-dir]
//
// Formats: openapi, asyncapi, d2, llms (default: openapi)
//
// This CLI generates from a built-in demo catalog. For real projects, import
// the catalog library directly and call the exporters programmatically.
package main

import (
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/openapi"
)

var errUnknownFormat = errors.New("unknown format")

const filePermUserOnly = 0o600

func main() {
	format := flag.String("format", "openapi", "output format: openapi, asyncapi, d2, llms")
	title := flag.String("title", "Demo API", "document title")
	version := flag.String("version", "1.0.0", "document version")
	output := flag.String("o", ".", "output directory")

	flag.Parse()

	cat := buildDemoCatalog(*title, *version)

	if err := generate(cat, *format, *output, *title, *version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	msg := "Generated " + *format + " documentation in " + *output
	_, _ = fmt.Fprintln(os.Stdout, msg)
}

func buildDemoCatalog(title, version string) *catalog.Catalog {
	reg := catalog.NewRegistry(title, version)
	reg.AddService(catalog.Service{
		ID:      "demo-svc",
		Name:    "Demo Service",
		Version: catalog.Version(version),
		Summary: "A demo service showing REST catalog features",
		Commands: []catalog.Message{
			{
				ID:      "item.create",
				Name:    "CreateItem",
				Version: "1.0.0",
				Kind:    catalog.CommandMessage,
				Operation: &catalog.Operation{
					Method:      "POST",
					Path:        "/api/items",
					StatusCodes: []string{"201", "400"},
				},
				Responses: []catalog.ResponseSpec{
					{StatusCode: "201", Description: "Item created"},
				},
			},
		},
		Queries: []catalog.Message{
			{
				ID:      "item.list",
				Name:    "ListItems",
				Version: "1.0.0",
				Kind:    catalog.QueryMessage,
				Operation: &catalog.Operation{
					Method: "GET",
					Path:   "/api/items",
				},
			},
		},
	})

	return reg.Build()
}

func generate(cat *catalog.Catalog, format, dir, title, version string) error {
	switch format {
	case "openapi":
		doc := openapi.NewExporter(title, version).Export(cat)

		return writeJSON(filepath.Join(dir, "openapi.json"), doc)
	case "asyncapi":
		doc := asyncapi.NewExporter(title, version).Export(cat)

		return writeJSON(filepath.Join(dir, "asyncapi.json"), doc)
	case "d2":
		output := d2.NewExporter(title, version).Export(cat)

		return os.WriteFile(filepath.Join(dir, "diagram.d2"), []byte(output), filePermUserOnly)
	case "llms":
		return writeLLMsTxt(filepath.Join(dir, "llms.txt"), cat)
	default:
		return fmt.Errorf("%w: %s", errUnknownFormat, format)
	}
}

func writeJSON(path string, v any) error {
	data, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return os.WriteFile(path, data, filePermUserOnly)
}

func writeLLMsTxt(path string, cat *catalog.Catalog) error {
	var builder strings.Builder

	fmt.Fprintf(&builder, "# %s\n\n> Auto-generated catalog summary.\n\n", cat.Title)

	for _, svc := range cat.Services {
		fmt.Fprintf(&builder, "## %s (%s)\n\n", svc.Name, svc.ID)

		for _, cmd := range svc.Commands {
			fmt.Fprintf(&builder, "- Command: %s", cmd.Name)

			if cmd.Operation != nil {
				fmt.Fprintf(&builder, " [%s %s]", cmd.Operation.Method, cmd.Operation.Path)
			}

			builder.WriteString("\n")
		}

		for _, q := range svc.Queries {
			fmt.Fprintf(&builder, "- Query: %s", q.Name)

			if q.Operation != nil {
				fmt.Fprintf(&builder, " [%s %s]", q.Operation.Method, q.Operation.Path)
			}

			builder.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(builder.String()), filePermUserOnly)
}
