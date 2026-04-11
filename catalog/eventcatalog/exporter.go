package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
)

type Exporter struct {
	OutputDir string
}

func NewExporter(outputDir string) *Exporter {
	return &Exporter{OutputDir: outputDir}
}

func (e *Exporter) Export(cat *catalog.Catalog) error {
	for _, svc := range cat.Services {
		err := e.writeService(svc)
		if err != nil {
			return fmt.Errorf("write service %s: %w", svc.ID, err)
		}

		for _, cmd := range svc.Commands {
			err := e.writeMessage(svc.ID, "commands", cmd)
			if err != nil {
				return fmt.Errorf("write command %s: %w", cmd.ID, err)
			}
		}

		for _, evt := range svc.Events {
			err := e.writeMessage(svc.ID, "events", evt)
			if err != nil {
				return fmt.Errorf("write event %s: %w", evt.ID, err)
			}
		}

		for _, q := range svc.Queries {
			err := e.writeMessage(svc.ID, "queries", q)
			if err != nil {
				return fmt.Errorf("write query %s: %w", q.ID, err)
			}
		}
	}

	for _, domain := range cat.Domains {
		err := e.writeDomain(domain)
		if err != nil {
			return fmt.Errorf("write domain %s: %w", domain.ID, err)
		}
	}

	return e.writeConfig(cat)
}

type frontmatterWriter struct {
	*strings.Builder
}

func newFrontmatterWriter() *frontmatterWriter {
	f := &frontmatterWriter{Builder: new(strings.Builder)}
	f.WriteString("---\n")

	return f
}

func (f *frontmatterWriter) addField(key, value string) {
	fmt.Fprintf(f, "%s: %s\n", key, value)
}

func (f *frontmatterWriter) addQuotedField(key, value string) {
	fmt.Fprintf(f, "%s: %q\n", key, value)
}

func (f *frontmatterWriter) addListField(key string, values []string) {
	if len(values) == 0 {
		return
	}

	fmt.Fprintf(f, "%s:\n", key)

	for _, v := range values {
		fmt.Fprintf(f, "  - %s\n", v)
	}
}

func (f *frontmatterWriter) finish(title, summary string) {
	f.WriteString("---\n\n")
	fmt.Fprintf(f, "# %s\n\n%s\n", title, summary)
}

func (e *Exporter) writeService(svc catalog.Service) error {
	dir := filepath.Join(e.OutputDir, "services", svc.ID)

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", svc.ID)
	md.addField("name", svc.Name)
	md.addField("version", svc.Version)

	if svc.Summary != "" {
		md.addQuotedField("summary", svc.Summary)
	}

	var sends, receives, commands, queries []string

	for _, msg := range svc.Events {
		id := messageID(msg)

		entry := fmt.Sprintf("%s/%s", id, msg.Version)
		if msg.Direction == catalog.Sends {
			sends = append(sends, entry)
		} else {
			receives = append(receives, entry)
		}
	}

	for _, cmd := range svc.Commands {
		commands = append(commands, fmt.Sprintf("%s/%s", messageID(cmd), cmd.Version))
	}

	for _, q := range svc.Queries {
		queries = append(queries, fmt.Sprintf("%s/%s", messageID(q), q.Version))
	}

	md.addListField("sends", sends)
	md.addListField("receives", receives)
	md.addListField("commands", commands)
	md.addListField("queries", queries)
	md.finish(svc.Name, svc.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func (e *Exporter) writeMessage(svcID, kind string, msg catalog.Message) error {
	id := messageID(msg)

	dir := filepath.Join(e.OutputDir, "services", svcID, kind, id)

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create message dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", id)
	md.addField("name", msg.Name)
	md.addField("version", msg.Version)

	if msg.Summary != "" {
		md.addQuotedField("summary", msg.Summary)
	}

	if msg.Schema != nil {
		md.WriteString("schemaPath: schemas/schema.json\n")
	}

	fmt.Fprintf(md, "owners:\n  - %s\n", svcID)
	md.finish(msg.Name, msg.Summary)

	if err := e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String()); err != nil {
		return fmt.Errorf("write message file: %w", err)
	}

	if msg.Schema != nil {
		return e.writeSchema(dir, msg.Schema)
	}

	return nil
}

func (e *Exporter) writeDomain(domain catalog.Domain) error {
	dir := filepath.Join(e.OutputDir, "domains", domain.ID)

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create domain dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", domain.ID)
	md.addField("name", domain.Name)
	md.addField("version", domain.Version)

	if domain.Summary != "" {
		md.addQuotedField("summary", domain.Summary)
	}

	md.addListField("services", domain.Services)
	md.finish(domain.Name, domain.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func (e *Exporter) writeMDXFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func (e *Exporter) writeSchema(dir string, schema *catalog.Schema) error {
	schemaDir := filepath.Join(dir, "schemas")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		return fmt.Errorf("create schema dir: %w", err)
	}

	data, err := catalog.SchemaToJSON(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	return os.WriteFile(filepath.Join(schemaDir, "schema.json"), data, 0o644)
}

func messageID(msg catalog.Message) string {
	return asyncapi.MessageID(msg)
}

func (e *Exporter) writeConfig(cat *catalog.Catalog) error {
	var cfg strings.Builder
	cfg.WriteString("/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */\n")
	cfg.WriteString("module.exports = {\n")
	fmt.Fprintf(&cfg, "  title: %q,\n", cat.Title)
	fmt.Fprintf(&cfg, "  organizationName: %q,\n", cat.Title)
	cfg.WriteString("  landingPage: { content: '' },\n")
	cfg.WriteString("};\n")

	return os.WriteFile(
		filepath.Join(e.OutputDir, "eventcatalog.config.js"),
		[]byte(cfg.String()),
		0o644,
	)
}
