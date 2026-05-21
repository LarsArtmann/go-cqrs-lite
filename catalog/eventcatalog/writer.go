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
		writeLLMsTxtService(&buf, svc)
	}

	return os.WriteFile( //nolint:wrapcheck // os.WriteFile returns direct error
		filepath.Join(e.outputDir, "llms.txt"),
		[]byte(buf.String()),
		filePerm,
	)
}

func writeLLMsTxtService(buf *strings.Builder, svc catalog.Service) {
	fmt.Fprintf(buf, "## %s (%s)\n", svc.Name, svc.ID)

	if svc.Summary != "" {
		fmt.Fprintf(buf, "%s\n", svc.Summary)
	}

	writeLLMsTxtMessages(buf, "Commands", svc.Commands)
	writeLLMsTxtEvents(buf, svc.Events)
	writeLLMsTxtMessages(buf, "Queries", svc.Queries)

	buf.WriteString("\n")
}

func writeLLMsTxtMessages(buf *strings.Builder, section string, msgs []catalog.Message) {
	if len(msgs) == 0 {
		return
	}

	fmt.Fprintf(buf, "\n### %s\n", section)

	for _, msg := range msgs {
		fmt.Fprintf(buf, "- %s (v%s): %s\n", msg.Name, msg.Version, msg.Summary)
	}
}

func writeLLMsTxtEvents(buf *strings.Builder, events []catalog.Message) {
	if len(events) == 0 {
		return
	}

	buf.WriteString("\n### Events\n")

	for _, evt := range events {
		dir := "receives"
		if evt.IsSend() {
			dir = "sends"
		}

		fmt.Fprintf(buf, "- %s (v%s) [%s]: %s\n", evt.Name, evt.Version, dir, evt.Summary)
	}
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

func (f *frontmatterWriter) addObjectIDsListField(key string, ids []string) {
	if len(ids) == 0 {
		return
	}

	_, _ = fmt.Fprintf(f, "%s:\n", key)

	for _, id := range ids {
		_, _ = fmt.Fprintf(f, "  - id: %s\n", id)
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
		return fmt.Errorf("create schema dir %s: %w", schemaDir, err)
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
	cfg.WriteString("  landingPage: '',\n")
	cfg.WriteString("};\n")

	err := os.WriteFile(
		filepath.Join(e.outputDir, "eventcatalog.config.js"),
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
		filepath.Join(e.outputDir, "package.json"),
		data,
		filePerm,
	)
}
