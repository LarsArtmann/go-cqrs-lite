package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// entitySection builds the section for one domain entity.
func entitySection(entity catalog.Entity, idx *refIndex) section {
	meta := []markup{ownerMeta(idx, entity.Owners)}

	if entity.AggregateRoot {
		meta = append(meta, textMeta("Aggregate root"))
	}

	if entity.Identifier != "" {
		meta = append(meta, textMeta("Identified by "+entity.Identifier))
	}

	sec := newSection(
		kindEntity,
		string(entity.ID),
		string(entity.Name),
	)

	sec.version = string(entity.Version)
	sec.summary = string(entity.Summary)

	sec.badges = entity.Badges
	sec.meta = metaLine(meta)

	sec.left = entityBlocks(entity)
	sec.right = entityRefs(entity, idx)

	return sec
}

// entityBlocks builds the left-column blocks of an entity section.
func entityBlocks(entity catalog.Entity) []block {
	blocks := make([]block, 0)

	if props := entityPropertyTable(entity.Properties); props != "" {
		blocks = append(blocks, blockHTML("Properties", props))
	}

	if table := attributeTable(entity.Schema); table != "" {
		blocks = append(blocks, blockHTML("Schema", table))
	}

	if raw := schemaJSON(entity.Schema); raw != "" {
		blocks = append(blocks, blockHTML("JSON Schema", codeBlock("", raw)))
	}

	return blocks
}

// entityRefs builds the relationship boxes of an entity section.
func entityRefs(entity catalog.Entity, idx *refIndex) []block {
	return appendRefs([]block{}, "Owners", idx.owners(entity.Owners))
}

// entityPropertyTable renders an entity's declared properties.
func entityPropertyTable(props []catalog.EntityProperty) markup {
	rows := make([][]tableCell, 0, len(props))

	for _, prop := range props {
		rows = append(rows, []tableCell{
			monoCell(string(prop.Name)),
			monoCell(prop.Type),
			plainCell(yesNo(prop.Required)),
			plainCell(prop.Description),
		})
	}

	return writeTable([]string{headerProperty, headerType, headerRequired, headerDescription}, rows)
}

// yesNo renders a boolean as a table cell value.
func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return ""
}

// dataProductSection builds the section for one data product.
func dataProductSection(product catalog.DataProduct, idx *refIndex) section {
	sec := newSection(
		kindDataProduct,
		string(product.ID),
		string(product.Name),
	)

	sec.version = string(product.Version)
	sec.summary = string(product.Summary)

	sec.badges = product.Badges
	sec.meta = metaLine([]markup{ownerMeta(idx, product.Owners)})

	sec.left = dataProductBlocks(product)
	sec.right = dataProductRefs(product, idx)

	return sec
}

// dataProductBlocks builds the left-column blocks of a data product section.
func dataProductBlocks(product catalog.DataProduct) []block {
	blocks := make([]block, 0)

	if outputs := outputTable(product.Outputs); outputs != "" {
		blocks = append(blocks, blockHTML("Outputs", outputs))
	}

	return blocks
}

// dataProductRefs builds the relationship boxes of a data product section.
func dataProductRefs(product catalog.DataProduct, idx *refIndex) []block {
	refs := make([]block, 0)
	refs = appendRefs(refs, "Inputs", idx.anyOf(refIDs(product.Inputs)))
	refs = appendRefs(refs, "Owners", idx.owners(product.Owners))

	return refs
}

// outputTable renders a data product's published outputs.
func outputTable(outputs []catalog.DataProductOutput) markup {
	rows := make([][]tableCell, 0, len(outputs))

	for _, output := range outputs {
		rows = append(rows, []tableCell{
			monoCell(string(output.ID)),
			plainCell(outputContractPath(output)),
			plainCell(outputContractType(output)),
		})
	}

	return writeTable([]string{"Output", "Contract", headerType}, rows)
}

// outputContractPath returns an output's contract path, if it declares one.
func outputContractPath(output catalog.DataProductOutput) string {
	if output.Contract == nil {
		return ""
	}

	return output.Contract.Path
}

// outputContractType returns an output's contract type, if it declares one.
func outputContractType(output catalog.DataProductOutput) string {
	if output.Contract == nil {
		return ""
	}

	return output.Contract.Type
}
