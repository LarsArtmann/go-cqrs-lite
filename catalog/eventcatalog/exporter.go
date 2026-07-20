package eventcatalog

import (
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

const (
	filePerm      = 0o600
	dirPerm       = 0o750
	indexFile     = "index.mdx"
	schemaPathKey = "schemas/schema.json"
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
func (e *Exporter) Export(cat *catalog.Catalog) error { //nolint:cyclop // straight-line pipeline
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

	for _, doc := range enriched.CustomDocs {
		err := e.writeCustomDoc(doc)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter.18",
				"write custom doc %s: %v",
				doc.ID,
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

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.12",
			"create service dir: %v", err)
	}

	sends, receives := collectSendsReceives(svc)

	fm := serviceFM{
		ID:             string(svc.ID),
		Name:           string(svc.Name),
		Version:        string(svc.Version),
		Summary:        string(svc.Summary),
		Owners:         svc.Owners,
		Sends:          sends,
		Receives:       receives,
		WritesTo:       toPointers(svc.WritesTo),
		ReadsFrom:      toPointers(svc.ReadsFrom),
		Entities:       svc.Entities,
		Flows:          stringIDsToStrings(svc.Flows),
		ExternalSystem: svc.ExternalSystem,
		Badges:         toBadges(svc.Badges),
		Repository:     toRepository(svc.Repository),
		Specifications: toSpecifications(svc.Specifications),
		Attachments:    toAttachments(svc.Attachments),
		baseConfigFM:   toBaseConfig(svc.BaseConfig),
	}

	content, err := renderMDX(fm, string(svc.Name), string(svc.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.12b",
			"render service %s: %v", svc.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}

func (e *Exporter) writeDomain(domain catalog.Domain) error {
	dir := filepath.Join(e.outputDir, "domains", string(domain.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.13",
			"create domain dir: %v", err)
	}

	domainIDs := make([]string, len(domain.SubDomains))
	for i, id := range domain.SubDomains {
		domainIDs[i] = string(id)
	}

	dpDataIDs := make([]string, len(domain.DataProducts))
	for i, id := range domain.DataProducts {
		dpDataIDs[i] = string(id)
	}

	fm := domainFM{
		ID:                 string(domain.ID),
		Name:               string(domain.Name),
		Version:            string(domain.Version),
		Summary:            string(domain.Summary),
		Owners:             domain.Owners,
		Services:           toPointers(domain.Services),
		Sends:              toRefs(domain.Sends),
		Receives:           toRefs(domain.Receives),
		Entities:           domain.Entities,
		Flows:              stringIDsToStrings(domain.Flows),
		Domains:            domainIDs,
		DataProducts:       dpDataIDs,
		UbiquitousLanguage: toUbiquitousLanguage(domain.UbiquitousLanguage),
		Badges:             toBadges(domain.Badges),
		Attachments:        toAttachments(domain.Attachments),
		baseConfigFM:       toBaseConfig(domain.BaseConfig),
	}

	content, err := renderMDX(fm, string(domain.Name), string(domain.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.13b",
			"render domain %s: %v", domain.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}
