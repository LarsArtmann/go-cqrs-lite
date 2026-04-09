package eventcatalog

import (
	"encoding/json"
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

func (e *Exporter) writeService(svc catalog.Service) error {
	dir := filepath.Join(e.OutputDir, "services", svc.ID)

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}

	var md strings.Builder
	md.WriteString("---\n")
	fmt.Fprintf(&md, "id: %s\n", svc.ID)
	fmt.Fprintf(&md, "name: %s\n", svc.Name)
	fmt.Fprintf(&md, "version: %s\n", svc.Version)

	if svc.Summary != "" {
		fmt.Fprintf(&md, "summary: %q\n", svc.Summary)
	}

	var sends, receives []string

	for _, msg := range svc.Events {
		id := messageID(msg)

		switch msg.Direction {
		case catalog.Sends:
			sends = append(sends, fmt.Sprintf("%s/%s", id, msg.Version))
		case catalog.Receives:
			receives = append(receives, fmt.Sprintf("%s/%s", id, msg.Version))
		}
	}

	if len(sends) > 0 {
		md.WriteString("sends:\n")

		for _, s := range sends {
			fmt.Fprintf(&md, "  - %s\n", s)
		}
	}

	if len(receives) > 0 {
		md.WriteString("receives:\n")

		for _, r := range receives {
			fmt.Fprintf(&md, "  - %s\n", r)
		}
	}

	md.WriteString("---\n\n")
	fmt.Fprintf(&md, "# %s\n\n%s\n", svc.Name, svc.Summary)

	return os.WriteFile(filepath.Join(dir, "index.mdx"), []byte(md.String()), 0o644)
}

func (e *Exporter) writeMessage(svcID, kind string, msg catalog.Message) error {
	id := messageID(msg)

	dir := filepath.Join(e.OutputDir, "services", svcID, kind, id)

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create message dir: %w", err)
	}

	var md strings.Builder
	md.WriteString("---\n")
	fmt.Fprintf(&md, "id: %s\n", id)
	fmt.Fprintf(&md, "name: %s\n", msg.Name)
	fmt.Fprintf(&md, "version: %s\n", msg.Version)

	if msg.Summary != "" {
		fmt.Fprintf(&md, "summary: %q\n", msg.Summary)
	}

	if msg.Schema != nil {
		md.WriteString("schemaPath: schemas/schema.json\n")
	}

	fmt.Fprintf(&md, "owners:\n  - %s\n", svcID)
	md.WriteString("---\n\n")
	fmt.Fprintf(&md, "# %s\n\n%s\n", msg.Name, msg.Summary)

	err = os.WriteFile(
		filepath.Join(dir, "index.mdx"),
		[]byte(md.String()),
		0o644,
	)
	if err != nil {
		return fmt.Errorf("write message file: %w", err)
	}

	if msg.Schema != nil {
		err = e.writeSchema(dir, msg.Schema)
		if err != nil {
			return fmt.Errorf("write schema: %w", err)
		}
	}

	return nil
}

func (e *Exporter) writeSchema(dir string, schema *catalog.Schema) error {
	schemaDir := filepath.Join(dir, "schemas")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		return fmt.Errorf("create schema dir: %w", err)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	return os.WriteFile(filepath.Join(schemaDir, "schema.json"), data, 0o644)
}

func (e *Exporter) writeDomain(domain catalog.Domain) error {
	dir := filepath.Join(e.OutputDir, "domains", domain.ID)

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create domain dir: %w", err)
	}

	var md strings.Builder
	md.WriteString("---\n")
	fmt.Fprintf(&md, "id: %s\n", domain.ID)
	fmt.Fprintf(&md, "name: %s\n", domain.Name)
	fmt.Fprintf(&md, "version: %s\n", domain.Version)

	if domain.Summary != "" {
		fmt.Fprintf(&md, "summary: %q\n", domain.Summary)
	}

	if len(domain.Services) > 0 {
		md.WriteString("services:\n")

		for _, s := range domain.Services {
			fmt.Fprintf(&md, "  - %s\n", s)
		}
	}

	md.WriteString("---\n\n")
	fmt.Fprintf(&md, "# %s\n\n%s\n", domain.Name, domain.Summary)

	return os.WriteFile(filepath.Join(dir, "index.mdx"), []byte(md.String()), 0o644)
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
