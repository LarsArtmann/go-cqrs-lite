package eventcatalog

import (
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	filePerm = 0o600
	dirPerm  = 0o750
	// eventCatalogCoreVersion pins the generated package.json dependency so
	// `npm install` in the output dir is reproducible. Bump deliberately
	// after verifying the new @eventcatalog/core release renders the
	// generated MDX correctly.
	eventCatalogCoreVersion = "^4.6.3"
	indexFile               = "index.mdx"
	schemaPathKey           = "schemas/schema.json"
)

// Exporter generates EventCatalog-compatible MDX files from a catalog.
type Exporter struct {
	outputDir string

	skipBootstrapFiles bool
	plainRefIDs         bool
}

// NewExporter creates an exporter that writes MDX files to the given output
// directory. Options (e.g. WithSkipBootstrapFiles) are applied in order.
func NewExporter(outputDir string, opts ...Option) *Exporter {
	e := &Exporter{outputDir: outputDir}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Export writes all services, messages, and schemas as MDX files to the output directory.
func (e *Exporter) Export(cat *catalog.Catalog) error { //nolint:cyclop,gocyclo // linear pipeline
	enriched := autoDeriveProducersConsumers(cat)

	for _, svc := range enriched.Services {
		if err := rejectUnsupportedBaseConfig(
			"service",
			string(svc.ID),
			svc.BaseConfig,
		); err != nil {
			return err
		}

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
	}

	if err := e.writeAllMessages(enriched); err != nil {
		return err
	}

	for _, domain := range enriched.Domains {
		if err := rejectUnsupportedBaseConfig(
			"domain",
			string(domain.ID),
			domain.BaseConfig,
		); err != nil {
			return err
		}

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
		err := e.writeChannel(ch, channelMessageIndex(enriched, e.plainRefIDs))
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

	if err := e.writeCoeffectSummary(cat); err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter.19",
			"write coeffects.md: %v",
			err,
		)
	}

	if err := e.writeSchemasTxt(cat); err != nil {
		return err
	}

	// The manifest is written last: it only exists for fully successful
	// exports, so a hub diffing catalog.index.json never sees a manifest
	// that describes a half-written tree.
	return e.writeIndexManifest(enriched)
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
		Entities:       toPointers(svc.Entities),
		Flows:          toPointers(svc.Flows),
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

	domainIDs := toPointers(domain.SubDomains)
	dpDataIDs := toPointers(domain.DataProducts)

	fm := domainFM{
		ID:           string(domain.ID),
		Name:         string(domain.Name),
		Version:      string(domain.Version),
		Summary:      string(domain.Summary),
		Owners:       domain.Owners,
		Services:     toPointers(domain.Services),
		Sends:        toRefs(domain.Sends),
		Receives:     toRefs(domain.Receives),
		Entities:     toPointers(domain.Entities),
		Flows:        toPointers(domain.Flows),
		Domains:      domainIDs,
		DataProducts: dpDataIDs,
		Badges:       toBadges(domain.Badges),
		Attachments:  toAttachments(domain.Attachments),
		baseConfigFM: toBaseConfig(domain.BaseConfig),
	}

	content, err := renderMDX(fm, string(domain.Name), string(domain.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.13b",
			"render domain %s: %v", domain.ID, err)
	}

	if err := e.writeMDXFile(filepath.Join(dir, indexFile), content); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.13c",
			"write domain %s: %v", domain.ID, err)
	}

	if err := e.writeUbiquitousLanguageFile(dir, domain.UbiquitousLanguage); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter.13d",
			"write ubiquitous language for %s: %v", domain.ID, err)
	}

	return nil
}
