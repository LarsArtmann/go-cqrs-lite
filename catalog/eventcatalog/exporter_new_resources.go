package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeEntity(entity catalog.Entity) error {
	dir := filepath.Join(e.outputDir, "entities", string(entity.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_new.1",
			"create entity dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	writeBaseFrontmatter(md, string(entity.ID), string(entity.Name), string(entity.Version),
		string(entity.Summary), entity.Owners)
	writeBadges(md, entity.Badges)

	if entity.Schema != nil {
		_, _ = md.WriteString("schemaPath: schemas/schema.json\n")
	}

	md.finishWithGraph(string(entity.Name), string(entity.Summary))

	err = e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_new.2",
			"write entity %s: %v",
			entity.ID,
			err,
		)
	}

	if entity.Schema != nil {
		return e.writeSchema(dir, entity.Schema)
	}

	return nil
}

func (e *Exporter) writeDataProduct(dp catalog.DataProduct) error {
	dir := filepath.Join(e.outputDir, "data-products", string(dp.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_new.3",
			"create data product dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	writeBaseFrontmatter(md, string(dp.ID), string(dp.Name), string(dp.Version),
		string(dp.Summary), dp.Owners)

	if dp.Domain != "" {
		md.addField("domain", string(dp.Domain))
	}

	writeBadges(md, dp.Badges)

	if dp.Schema != nil {
		_, _ = md.WriteString("schemaPath: schemas/schema.json\n")
	}

	md.finishWithGraph(string(dp.Name), string(dp.Summary))

	err = e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_new.4",
			"write data product %s: %v",
			dp.ID,
			err,
		)
	}

	if dp.Schema != nil {
		return e.writeSchema(dir, dp.Schema)
	}

	return nil
}

func (e *Exporter) writeAgent(agent catalog.Agent) error {
	dir := filepath.Join(e.outputDir, "agents", string(agent.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_new.5",
			"create agent dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	writeBaseFrontmatter(md, string(agent.ID), string(agent.Name), string(agent.Version),
		string(agent.Summary), agent.Owners)

	sends, receives, commands, queries := collectAgentMessageIDs(agent)
	addObjectIDsListField(md, "sends", sends)
	addObjectIDsListField(md, "receives", receives)
	addObjectIDsListField(md, "commands", commands)
	addObjectIDsListField(md, "queries", queries)
	writeIDListField(md, "dataStores", agent.DataStores)
	writeIDListField(md, "flows", agent.Flows)
	writeBadges(md, agent.Badges)
	md.finishWithGraph(string(agent.Name), string(agent.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
}

func collectAgentMessageIDs(
	agent catalog.Agent,
) (sends, receives, commands, queries []catalog.MessageID) {
	sends = make([]catalog.MessageID, 0, len(agent.Events))
	receives = make([]catalog.MessageID, 0, len(agent.Events))
	commands = make([]catalog.MessageID, 0, len(agent.Commands))
	queries = make([]catalog.MessageID, 0, len(agent.Queries))

	for _, msg := range agent.Events {
		messageID := catalog.Key(msg)
		if msg.IsSend() {
			sends = append(sends, messageID)
		} else {
			receives = append(receives, messageID)
		}
	}

	for _, cmd := range agent.Commands {
		commands = append(commands, catalog.Key(cmd))
	}

	for _, q := range agent.Queries {
		queries = append(queries, catalog.Key(q))
	}

	return sends, receives, commands, queries
}

func writeUbiquitousLanguage(md *frontmatterWriter, terms []catalog.UbiquitousLanguageTerm) {
	if len(terms) == 0 {
		return
	}

	_, _ = md.WriteString("ubiquitousLanguage:\n")

	for _, t := range terms {
		_, _ = fmt.Fprintf(md, "  - name: %q\n", t.Name)
		if t.Description != "" {
			_, _ = fmt.Fprintf(md, "    description: %q\n", t.Description)
		}
	}
}
