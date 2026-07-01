package eventcatalog

import (
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeEntity(entity catalog.Entity) error {
	dir := filepath.Join(e.outputDir, "entities", string(entity.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.1",
			"create entity dir: %v", err)
	}

	fm := entityFM{
		ID:            string(entity.ID),
		Name:          string(entity.Name),
		Version:       string(entity.Version),
		Summary:       string(entity.Summary),
		AggregateRoot: entity.AggregateRoot,
		Identifier:    entity.Identifier,
		Properties:    toEntityProperties(entity.Properties),
		Owners:        entity.Owners,
		Badges:        toBadges(entity.Badges),
		Schemas:       toSchemas(entity.Schemas),
	}

	if entity.Schema != nil {
		fm.SchemaPath = "schemas/schema.json"
	}

	content, err := renderMDX(fm, string(entity.Name), string(entity.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.2",
			"render entity %s: %v", entity.ID, err)
	}

	if err := e.writeMDXFile(filepath.Join(dir, indexFile), content); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.2b",
			"write entity %s: %v", entity.ID, err)
	}

	if entity.Schema != nil {
		return e.writeSchema(dir, entity.Schema)
	}

	return nil
}

func (e *Exporter) writeDataProduct(dp catalog.DataProduct) error {
	dir := filepath.Join(e.outputDir, "data-products", string(dp.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.3",
			"create data product dir: %v", err)
	}

	fm := dataProductFM{
		ID:      string(dp.ID),
		Name:    string(dp.Name),
		Version: string(dp.Version),
		Summary: string(dp.Summary),
		Hidden:  dp.Hidden,
		Owners:  dp.Owners,
		Inputs:  toRefs(dp.Inputs),
		Outputs: toDataProductOutputs(dp.Outputs),
		Badges:  toBadges(dp.Badges),
	}

	content, err := renderMDX(fm, string(dp.Name), string(dp.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.4",
			"render data product %s: %v", dp.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}

func (e *Exporter) writeAgent(agent catalog.Agent) error {
	dir := filepath.Join(e.outputDir, "agents", string(agent.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.5",
			"create agent dir: %v", err)
	}

	fm := agentFM{
		ID:        string(agent.ID),
		Name:      string(agent.Name),
		Version:   string(agent.Version),
		Summary:   string(agent.Summary),
		Owners:    agent.Owners,
		Sends:     toRefs(agent.Sends),
		Receives:  toRefs(agent.Receives),
		ReadsFrom: toPointers(agent.ReadsFrom),
		WritesTo:  toPointers(agent.WritesTo),
		Model:     toAgentModel(agent.Model),
		Tools:     toAgentTools(agent.Tools),
		Flows:     flowIDsToStrings(agent.Flows),
		Badges:    toBadges(agent.Badges),
	}

	content, err := renderMDX(fm, string(agent.Name), string(agent.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_new.6",
			"render agent %s: %v", agent.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}
