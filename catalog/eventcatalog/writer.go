package eventcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func (e *Exporter) writeLLMsTxt(cat *catalog.Catalog) error {
	var buf strings.Builder

	fmt.Fprintf(&buf, "# %s\n\n", cat.Title)
	buf.WriteString("> Auto-generated catalog summary for LLM consumption.\n\n")

	for _, svc := range cat.Services {
		fmt.Fprintf(&buf, "## %s (%s)\n", svc.Name, svc.ID)

		if svc.Summary != "" {
			fmt.Fprintf(&buf, "%s\n", svc.Summary)
		}

		if len(svc.Commands) > 0 {
			buf.WriteString("\n### Commands\n")

			for _, cmd := range svc.Commands {
				fmt.Fprintf(&buf, "- %s (v%s): %s\n", cmd.Name, cmd.Version, cmd.Summary)
			}
		}

		if len(svc.Events) > 0 {
			buf.WriteString("\n### Events\n")

			for _, evt := range svc.Events {
				dir := "receives"
				if evt.Direction == catalog.Sends {
					dir = "sends"
				}

				fmt.Fprintf(&buf, "- %s (v%s) [%s]: %s\n", evt.Name, evt.Version, dir, evt.Summary)
			}
		}

		if len(svc.Queries) > 0 {
			buf.WriteString("\n### Queries\n")

			for _, q := range svc.Queries {
				fmt.Fprintf(&buf, "- %s (v%s): %s\n", q.Name, q.Version, q.Summary)
			}
		}

		buf.WriteString("\n")
	}

	return os.WriteFile( //nolint:wrapcheck // os.WriteFile returns direct error
		filepath.Join(e.OutputDir, "llms.txt"),
		[]byte(buf.String()),
		filePerm,
	)
}

type frontmatterWriter struct {
	*strings.Builder
}

func newFrontmatterWriter() *frontmatterWriter {
	f := &frontmatterWriter{Builder: new(strings.Builder)}
	_, _ = f.WriteString("---\n")

	return f
}

func (f *frontmatterWriter) addField(key, value string) {
	_, _ = fmt.Fprintf(f, "%s: %s\n", key, value)
}

func (f *frontmatterWriter) addQuotedField(key, value string) {
	_, _ = fmt.Fprintf(f, "%s: %q\n", key, value)
}

func (f *frontmatterWriter) addListField(key string, values []string) {
	if len(values) == 0 {
		return
	}

	_, _ = fmt.Fprintf(f, "%s:\n", key)

	for _, v := range values {
		_, _ = fmt.Fprintf(f, "  - %s\n", v)
	}
}

func (f *frontmatterWriter) finish(title, summary string) {
	_, _ = f.WriteString("---\n\n")
	_, _ = fmt.Fprintf(f, "# %s\n\n%s\n", title, summary)
}

func (e *Exporter) writeMDXFile(path, content string) error {
	return os.WriteFile(path, []byte(content), filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeSchema(dir string, schema *catalog.Schema) error {
	schemaDir := filepath.Join(dir, "schemas")

	err := os.MkdirAll(schemaDir, dirPerm)
	if err != nil {
		return fmt.Errorf("create schema dir: %w", err)
	}

	data, err := catalog.SchemaToJSON(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	return os.WriteFile(filepath.Join(schemaDir, "schema.json"), data, filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeExamples(dir string, examples []json.RawMessage) error {
	if len(examples) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal examples: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, "examples.json"), data, filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeConfig(cat *catalog.Catalog) error {
	var cfg strings.Builder
	cfg.WriteString("/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */\n")
	cfg.WriteString("export default {\n")
	fmt.Fprintf(&cfg, "  title: %q,\n", cat.Title)
	fmt.Fprintf(&cfg, "  organizationName: %q,\n", cat.Title)
	cfg.WriteString("  landingPage: { content: '' },\n")
	cfg.WriteString("};\n")

	err := os.WriteFile(
		filepath.Join(e.OutputDir, "eventcatalog.config.js"),
		[]byte(cfg.String()),
		filePerm,
	)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return e.writePackageJSON(cat)
}

func (e *Exporter) writePackageJSON(cat *catalog.Catalog) error {
	pkg := map[string]any{
		"type":        "module",
		"name":        strings.ToLower(strings.ReplaceAll(cat.Title, " ", "-")),
		"version":     cat.Version,
		"private":     true,
		"description": cat.Title + " event catalog",
		"dependencies": map[string]string{
			"@eventcatalog/core": "latest",
		},
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}

	return os.WriteFile( //nolint:wrapcheck
		filepath.Join(e.OutputDir, "package.json"),
		data,
		filePerm,
	)
}
