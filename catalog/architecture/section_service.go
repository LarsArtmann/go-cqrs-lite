package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// serviceSection builds the section for one service.
func serviceSection(service catalog.Service, idx *refIndex) section {
	meta := []markup{
		ownerMeta(idx, service.Owners),
		countMeta(len(service.Commands)+len(service.Events)+len(service.Queries), "message", "messages"),
	}

	if service.ExternalSystem {
		meta = append(meta, textMeta("External system"))
	}

	return section{
		kind:    kindService,
		anchor:  idx.get(kindService, string(service.ID)).anchor,
		name:    string(service.Name),
		version: string(service.Version),
		summary: string(service.Summary),
		badges:  service.Badges,
		meta:    metaLine(meta),
		left:    serviceBlocks(service, idx),
		right:   serviceRefs(service, idx),
	}
}

// serviceBlocks builds the left-column blocks of a service section.
func serviceBlocks(service catalog.Service, idx *refIndex) []block {
	blocks := make([]block, 0, 3)

	blocks = appendRefs(blocks, "Publishes", serviceMessages(service, idx, catalog.Sends))
	blocks = appendRefs(blocks, "Handles", serviceMessages(service, idx, catalog.Receives))

	if links := attachmentLinks(service.Attachments); links != "" {
		blocks = append(blocks, blockHTML("Attachments", links))
	}

	return blocks
}

// serviceRefs builds the relationship boxes of a service section.
func serviceRefs(service catalog.Service, idx *refIndex) []block {
	refs := make([]block, 0, 6)
	refs = appendRefs(refs, "Writes to", idx.messagesOf(kindDataStore, idStrings(service.WritesTo)))
	refs = appendRefs(refs, "Reads from", idx.messagesOf(kindDataStore, idStrings(service.ReadsFrom)))
	refs = appendRefs(refs, "Entities", idx.messagesOf(kindEntity, service.Entities))
	refs = appendRefs(refs, "Flows", idx.messagesOf(kindFlow, idStrings(service.Flows)))

	if specs := specList(service.Specifications); specs != "" {
		refs = append(refs, blockCounted("Specifications", len(service.Specifications), specs))
	}

	if service.Repository != nil && service.Repository.URL != "" {
		refs = append(refs, blockHTML("Repository", externalLink(string(service.Repository.URL), "Source code")))
	}

	return refs
}

// serviceMessages resolves the service's messages of one direction, keeping
// the message kind on each reference so the reader sees what it is.
func serviceMessages(service catalog.Service, idx *refIndex, direction catalog.Direction) []resourceRef {
	refs := make([]resourceRef, 0, len(service.Events))

	for _, kind := range []resourceKind{kindCommand, kindEvent, kindQuery} {
		for _, msg := range messagesOf(service, kind) {
			if msg.Direction != direction {
				continue
			}

			ref := idx.get(kind, string(msg.ID))
			ref.note = string(msg.Kind)

			refs = append(refs, ref)
		}
	}

	return refs
}

// specList renders a service's machine-readable specifications as rows.
func specList(specs []catalog.Specification) markup {
	if len(specs) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		for _, spec := range specs {
			w.open("div", htmlClass("spec-row"))

			if name := string(spec.Name); name != "" {
				w.element("span", name, htmlClass("spec-name"))
			}

			w.element("span", spec.Type, htmlClass("spec-type"))
			w.element("span", spec.Path, htmlClass("spec-path"))
			w.close("div")
		}
	})
}
