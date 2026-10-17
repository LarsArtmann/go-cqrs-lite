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
	if len(props) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md-table"))
		w.open("table")
		w.open("thead")
		w.open("tr")

		for _, header := range []string{headerProperty, headerType, headerRequired, headerDescription} {
			w.element("th", header)
		}

		w.close("tr")
		w.close("thead")
		w.open("tbody")

		for _, prop := range props {
			w.open("tr")
			w.element("td", string(prop.Name), htmlClass("mono"))
			w.element("td", prop.Type, htmlClass("mono"))
			w.element("td", yesNo(prop.Required))
			w.element("td", prop.Description)
			w.close("tr")
		}

		w.close("tbody")
		w.close("table")
		w.close("div")
	})
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
	if len(outputs) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md-table"))
		w.open("table")
		w.open("thead")
		w.open("tr")

		for _, header := range []string{"Output", "Contract", headerType} {
			w.element("th", header)
		}

		w.close("tr")
		w.close("thead")
		w.open("tbody")

		for _, output := range outputs {
			w.open("tr")
			w.element("td", string(output.ID), htmlClass("mono"))
			w.element("td", outputContractPath(output))
			w.element("td", outputContractType(output))
			w.close("tr")
		}

		w.close("tbody")
		w.close("table")
		w.close("div")
	})
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
