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

	writeMessagePointers(md, "inputs", dp.Inputs)
	writeMessagePointers(md, "outputs", dp.Outputs)
	writeBadges(md, dp.Badges)
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

	writeMessagePointers(md, "sends", agent.Sends)
	writeMessagePointers(md, "receives", agent.Receives)
	writeIDListField(md, "readsFrom", agent.ReadsFrom)
	writeIDListField(md, "writesTo", agent.WritesTo)
	writeAgentModel(md, agent.Model)
	writeAgentTools(md, agent.Tools)
	writeIDListField(md, "flows", agent.Flows)
	writeBadges(md, agent.Badges)
	md.finishWithGraph(string(agent.Name), string(agent.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
}

func writeAgentModel(md *frontmatterWriter, model *catalog.AgentModel) {
	if model == nil {
		return
	}

	_, _ = md.WriteString("model:\n")
	_, _ = fmt.Fprintf(md, "  provider: %q\n", model.Provider)
	_, _ = fmt.Fprintf(md, "  name: %q\n", model.Name)

	if model.Version != "" {
		_, _ = fmt.Fprintf(md, "  version: %q\n", model.Version)
	}
}

func writeAgentTools(md *frontmatterWriter, tools []catalog.AgentTool) {
	if len(tools) == 0 {
		return
	}

	_, _ = md.WriteString("tools:\n")

	for _, t := range tools {
		_, _ = fmt.Fprintf(md, "  - name: %q\n", t.Name)

		if t.Type != "" {
			_, _ = fmt.Fprintf(md, "    type: %s\n", t.Type)
		}

		if t.URL != "" {
			_, _ = fmt.Fprintf(md, "    url: %q\n", t.URL)
		}

		if t.Description != "" {
			_, _ = fmt.Fprintf(md, "    description: %q\n", t.Description)
		}

		if t.Icon != "" {
			_, _ = fmt.Fprintf(md, "    icon: %q\n", t.Icon)
		}
	}
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
