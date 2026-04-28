package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	filePerm = 0o600
	dirPerm  = 0o750
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

	err := e.writeConfig(cat)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return e.writeLLMsTxt(cat)
}

func (e *Exporter) writeService(svc catalog.Service) error {
	dir := filepath.Join(e.OutputDir, "services", svc.ID)

	err := os.MkdirAll(dir, dirPerm)
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

	md.addListField("owners", svc.Owners)

	var sends, receives, commands, queries []string

	for _, msg := range svc.Events {
		id := catalog.MessageID(msg)

		entry := fmt.Sprintf("%s/%s", id, msg.Version)
		if msg.Direction == catalog.Sends {
			sends = append(sends, entry)
		} else {
			receives = append(receives, entry)
		}
	}

	for _, cmd := range svc.Commands {
		commands = append(commands, fmt.Sprintf("%s/%s", catalog.MessageID(cmd), cmd.Version))
	}

	for _, q := range svc.Queries {
		queries = append(queries, fmt.Sprintf("%s/%s", catalog.MessageID(q), q.Version))
	}

	md.addListField("sends", sends)
	md.addListField("receives", receives)
	md.addListField("commands", commands)
	md.addListField("queries", queries)
	md.finish(svc.Name, svc.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func (e *Exporter) writeMessage(svcID, kind string, msg catalog.Message) error {
	id := catalog.MessageID(msg)

	dir := filepath.Join(e.OutputDir, "services", svcID, kind, id)

	err := os.MkdirAll(dir, dirPerm)
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
		_, _ = md.WriteString("schemaPath: schemas/schema.json\n")
	}

	_, _ = fmt.Fprintf(md, "owners:\n  - %s\n", svcID)
	md.finish(msg.Name, msg.Summary)

	err = e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
	if err != nil {
		return fmt.Errorf("write message file: %w", err)
	}

	if msg.Schema != nil {
		err := e.writeSchema(dir, msg.Schema)
		if err != nil {
			return err
		}
	}

	return e.writeExamples(dir, msg.Examples)
}

func (e *Exporter) writeDomain(domain catalog.Domain) error {
	dir := filepath.Join(e.OutputDir, "domains", domain.ID)

	err := os.MkdirAll(dir, dirPerm)
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
