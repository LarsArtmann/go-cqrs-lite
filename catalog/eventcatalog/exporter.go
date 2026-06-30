package eventcatalog

import (
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

const (
	filePerm  = 0o600
	dirPerm   = 0o750
	indexFile = "index.mdx"
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
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.1",
				"write service %s: %v",
				svc.ID,
				err,
			)
		}

		err = e.writeServiceMessages(svc)
		if err != nil {
			return err
		}
	}

	for _, domain := range enriched.Domains {
		err := e.writeDomain(domain)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.2",
				"write domain %s: %v",
				domain.ID,
				err,
			)
		}
	}

	for _, ch := range enriched.Channels {
		err := e.writeChannel(ch)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.3",
				"write channel %s: %v",
				ch.ID,
				err,
			)
		}
	}

	for _, ds := range enriched.DataStores {
		err := e.writeDataStore(ds)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.4",
				"write data store %s: %v",
				ds.ID,
				err,
			)
		}
	}

	for _, f := range enriched.Flows {
		err := e.writeFlow(f)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.5",
				"write flow %s: %v",
				f.ID,
				err,
			)
		}
	}

	for _, team := range enriched.Teams {
		err := e.writeTeam(team)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.6",
				"write team %s: %v",
				team.ID,
				err,
			)
		}
	}

	for _, user := range enriched.Users {
		err := e.writeUser(user)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.7",
				"write user %s: %v",
				user.ID,
				err,
			)
		}
	}

	for _, entity := range enriched.Entities {
		err := e.writeEntity(entity)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.14",
				"write entity %s: %v",
				entity.ID,
				err,
			)
		}
	}

	for _, dp := range enriched.DataProducts {
		err := e.writeDataProduct(dp)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.15",
				"write data product %s: %v",
				dp.ID,
				err,
			)
		}
	}

	for _, agent := range enriched.Agents {
		err := e.writeAgent(agent)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.16",
				"write agent %s: %v",
				agent.ID,
				err,
			)
		}
	}

	err := e.writeConfig(cat)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter.8",
			"write config: %v",
			err,
		)
	}

	if err := e.writeLLMsTxt(cat); err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter.17",
			"write llms.txt: %v",
			err,
		)
	}

	return e.writeSchemasTxt(cat)
}

func (e *Exporter) writeServiceMessages(svc catalog.Service) error {
	serviceID := svc.ID

	for _, cmd := range svc.Commands {
		err := e.writeMessage(serviceID, "commands", cmd)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.9",
				"write command %s: %v",
				cmd.ID,
				err,
			)
		}
	}

	for _, evt := range svc.Events {
		err := e.writeMessage(serviceID, "events", evt)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.10",
				"write event %s: %v",
				evt.ID,
				err,
			)
		}
	}

	for _, q := range svc.Queries {
		err := e.writeMessage(serviceID, "queries", q)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.11",
				"write query %s: %v",
				q.ID,
				err,
			)
		}
	}

	return nil
}

func (e *Exporter) writeService(svc catalog.Service) error {
	dir := filepath.Join(e.outputDir, "services", string(svc.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter.12",
			"create service dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	writeBaseFrontmatter(md, string(svc.ID), string(svc.Name), string(svc.Version),
		string(svc.Summary), svc.Owners)

	sends, receives, commands, queries := collectMessageIDs(svc)

	addObjectIDsListField(md, "sends", sends)
	addObjectIDsListField(md, "receives", receives)
	addObjectIDsListField(md, "commands", commands)
	addObjectIDsListField(md, "queries", queries)
	writeIDListField(md, "writesTo", svc.WritesTo)
	writeIDListField(md, "readsFrom", svc.ReadsFrom)
	writeIDListField(md, "entities", svc.Entities)
	writeIDListField(md, "flows", svc.Flows)

	if svc.ExternalSystem {
		md.addField("externalSystem", "true")
	}

	writeBadges(md, svc.Badges)
	writeRepository(md, svc.Repository)
	writeSpecifications(md, svc.Specifications)
	writeAttachments(md, svc.Attachments)
	md.finishWithGraph(string(svc.Name), string(svc.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
}

func collectMessageIDs(
	svc catalog.Service,
) (sends, receives, commands, queries []catalog.MessageID) {
	sends = make([]catalog.MessageID, 0, len(svc.Events))
	receives = make([]catalog.MessageID, 0, len(svc.Events))
	commands = make([]catalog.MessageID, 0, len(svc.Commands))
	queries = make([]catalog.MessageID, 0, len(svc.Queries))

	for _, msg := range svc.Events {
		messageID := catalog.Key(msg)

		if msg.IsSend() {
			sends = append(sends, messageID)
		} else {
			receives = append(receives, messageID)
		}
	}

	for _, cmd := range svc.Commands {
		commands = append(commands, catalog.Key(cmd))
	}

	for _, q := range svc.Queries {
		queries = append(queries, catalog.Key(q))
	}

	return sends, receives, commands, queries
}

func (e *Exporter) writeDomain(domain catalog.Domain) error {
	dir := filepath.Join(e.outputDir, "domains", string(domain.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter.13",
			"create domain dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	writeBaseFrontmatter(md, string(domain.ID), string(domain.Name), string(domain.Version),
		string(domain.Summary), domain.Owners)

	addObjectIDsListField(md, "services", domain.Services)
	writeMessagePointers(md, "sends", domain.Sends)
	writeMessagePointers(md, "receives", domain.Receives)
	writeIDListField(md, "entities", domain.Entities)
	writeIDListField(md, "flows", domain.Flows)
	writeIDListField(md, "subDomains", domain.SubDomains)
	writeIDListField(md, "dataProducts", domain.DataProducts)
	writeUbiquitousLanguage(md, domain.UbiquitousLanguage)
	writeBadges(md, domain.Badges)
	writeAttachments(md, domain.Attachments)
	md.finishWithGraph(string(domain.Name), string(domain.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
}
