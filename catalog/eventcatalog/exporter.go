package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	filePerm = 0o600
	dirPerm  = 0o750
)

// Exporter generates EventCatalog-compatible MDX files from a catalog.
type Exporter struct {
	outputDir string
}

// NewExporter creates an exporter that writes MDX files to the given output directory.
func NewExporter(outputDir string) *Exporter {
	return &Exporter{outputDir: outputDir}
}

// Export writes all services, messages, and schemas as MDX files to the output directory.
func (e *Exporter) Export(cat *catalog.Catalog) error {
	enriched := autoDeriveProducersConsumers(cat)

	for _, svc := range enriched.Services {
		err := e.writeService(svc)
		if err != nil {
			return fmt.Errorf("write service %s: %w", svc.ID, err)
		}

		err = e.writeServiceMessages(svc)
		if err != nil {
			return err
		}
	}

	for _, domain := range enriched.Domains {
		err := e.writeDomain(domain)
		if err != nil {
			return fmt.Errorf("write domain %s: %w", domain.ID, err)
		}
	}

	for _, ch := range enriched.Channels {
		err := e.writeChannel(ch)
		if err != nil {
			return fmt.Errorf("write channel %s: %w", ch.ID, err)
		}
	}

	for _, ds := range enriched.DataStores {
		err := e.writeDataStore(ds)
		if err != nil {
			return fmt.Errorf("write data store %s: %w", ds.ID, err)
		}
	}

	for _, f := range enriched.Flows {
		err := e.writeFlow(f)
		if err != nil {
			return fmt.Errorf("write flow %s: %w", f.ID, err)
		}
	}

	for _, team := range enriched.Teams {
		err := e.writeTeam(team)
		if err != nil {
			return fmt.Errorf("write team %s: %w", team.ID, err)
		}
	}

	for _, user := range enriched.Users {
		err := e.writeUser(user)
		if err != nil {
			return fmt.Errorf("write user %s: %w", user.ID, err)
		}
	}

	err := e.writeConfig(cat)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return e.writeLLMsTxt(cat)
}

func (e *Exporter) writeServiceMessages(svc catalog.Service) error {
	svcID := svc.ID

	for _, cmd := range svc.Commands {
		err := e.writeMessage(svcID, "commands", cmd)
		if err != nil {
			return fmt.Errorf("write command %s: %w", cmd.ID, err)
		}
	}

	for _, evt := range svc.Events {
		err := e.writeMessage(svcID, "events", evt)
		if err != nil {
			return fmt.Errorf("write event %s: %w", evt.ID, err)
		}
	}

	for _, q := range svc.Queries {
		err := e.writeMessage(svcID, "queries", q)
		if err != nil {
			return fmt.Errorf("write query %s: %w", q.ID, err)
		}
	}

	return nil
}

func (e *Exporter) writeService(svc catalog.Service) error {
	dir := filepath.Join(e.outputDir, "services", string(svc.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(svc.ID))
	md.addField("name", svc.Name)
	md.addField("version", svc.Version)

	if svc.Summary != "" {
		md.addQuotedField("summary", svc.Summary)
	}

	md.addListField("owners", svc.Owners)

	sends, receives, commands, queries := collectMessageIDs(svc)

	md.addObjectIDsListField("sends", sends)
	md.addObjectIDsListField("receives", receives)
	md.addObjectIDsListField("commands", commands)
	md.addObjectIDsListField("queries", queries)
	writeIDListField(md, "writesTo", svc.WritesTo)
	writeIDListField(md, "readsFrom", svc.ReadsFrom)
	writeIDListField(md, "entities", svc.Entities)
	writeIDListField(md, "flows", svc.Flows)
	writeBadges(md, svc.Badges)
	writeRepository(md, svc.Repository)
	writeSpecifications(md, svc.Specifications)
	writeAttachments(md, svc.Attachments)
	md.finishWithGraph(svc.Name, svc.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func collectMessageIDs(svc catalog.Service) ([]string, []string, []string, []string) {
	sends := make([]string, 0, len(svc.Events))
	receives := make([]string, 0, len(svc.Events))
	commands := make([]string, 0, len(svc.Commands))
	queries := make([]string, 0, len(svc.Queries))

	for _, msg := range svc.Events {
		id := string(catalog.GetID(msg))

		if msg.IsSend() {
			sends = append(sends, id)
		} else {
			receives = append(receives, id)
		}
	}

	for _, cmd := range svc.Commands {
		commands = append(commands, string(catalog.GetID(cmd)))
	}

	for _, q := range svc.Queries {
		queries = append(queries, string(catalog.GetID(q)))
	}

	return sends, receives, commands, queries
}

func (e *Exporter) writeMessage(svcID catalog.ServiceID, kind string, msg catalog.Message) error {
	id := string(catalog.GetID(msg))
	dir := filepath.Join(e.outputDir, "services", string(svcID), kind, id)

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create message dir for %s/%s: %w", svcID, kind, err)
	}

	md := buildMessageFrontmatter(id, msg)

	err = e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
	if err != nil {
		return fmt.Errorf("write message file for %s/%s: %w", svcID, kind, err)
	}

	if msg.Schema != nil {
		err = e.writeSchema(dir, msg.Schema)
		if err != nil {
			return fmt.Errorf("write schema for %s/%s: %w", svcID, kind, err)
		}
	}

	return e.writeExamples(dir, msg.Examples)
}

func buildMessageFrontmatter(id string, msg catalog.Message) *frontmatterWriter {
	md := newFrontmatterWriter()
	md.addField("id", id)
	md.addField("name", msg.Name)
	md.addField("version", msg.Version)

	if msg.Summary != "" {
		md.addQuotedField("summary", msg.Summary)
	}

	if msg.Deprecated {
		md.addField("deprecated", "true")
	}

	md.addListField("owners", msg.Owners)
	writeLabels(md, msg.Labels)
	writeChangelog(md, msg.Changelog)
	writeIDListField(md, "producers", msg.Producers)
	writeIDListField(md, "consumers", msg.Consumers)
	writeOperation(md, msg.Operation)
	writeBadges(md, msg.Badges)
	writeRepository(md, msg.Repository)

	if msg.Schema != nil {
		_, _ = md.WriteString("schemaPath: schemas/schema.json\n")
	}

	md.finish(msg.Name, msg.Summary)

	return md
}

func writeLabels(md *frontmatterWriter, labels map[string]string) {
	if len(labels) == 0 {
		return
	}

	_, _ = md.WriteString("labels:\n")

	for k, v := range labels {
		_, _ = fmt.Fprintf(md, "  %s: %q\n", k, v)
	}
}

func writeChangelog(md *frontmatterWriter, changelog []catalog.Change) {
	if len(changelog) == 0 {
		return
	}

	_, _ = md.WriteString("changelog:\n")

	for _, c := range changelog {
		_, _ = fmt.Fprintf(md, "  - version: %q\n    summary: %q", c.Version, c.Summary)

		if c.Date != nil {
			_, _ = fmt.Fprintf(md, "\n    date: %q", c.Date.Format(time.DateOnly))
		}

		_, _ = md.WriteString("\n")
	}
}

func (e *Exporter) writeDomain(domain catalog.Domain) error {
	dir := filepath.Join(e.outputDir, "domains", string(domain.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create domain dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(domain.ID))
	md.addField("name", domain.Name)
	md.addField("version", domain.Version)

	if domain.Summary != "" {
		md.addQuotedField("summary", domain.Summary)
	}

	md.addListField("owners", domain.Owners)

	strs := make([]string, len(domain.Services))
	for i, s := range domain.Services {
		strs[i] = string(s)
	}

	md.addObjectIDsListField("services", strs)
	writeMessagePointers(md, "sends", domain.Sends)
	writeMessagePointers(md, "receives", domain.Receives)
	writeIDListField(md, "entities", domain.Entities)
	writeIDListField(md, "flows", domain.Flows)
	writeBadges(md, domain.Badges)
	writeAttachments(md, domain.Attachments)
	md.finishWithGraph(domain.Name, domain.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}
