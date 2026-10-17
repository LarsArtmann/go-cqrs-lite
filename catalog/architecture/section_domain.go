package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// domainSection builds the section for one domain.
func domainSection(domain catalog.Domain, idx *refIndex) section {
	return section{
		kind:    kindDomain,
		anchor:  idx.get(kindDomain, string(domain.ID)).anchor,
		name:    string(domain.Name),
		version: string(domain.Version),
		summary: string(domain.Summary),
		badges:  domain.Badges,
		meta: metaLine([]markup{
			ownerMeta(idx, domain.Owners),
			countMeta(len(domain.Services), "service", "services"),
			countMeta(len(domain.SubDomains), "sub-domain", "sub-domains"),
		}),
		left:  domainBlocks(domain),
		right: domainRefs(domain, idx),
	}
}

// domainBlocks builds the left-column blocks of a domain section.
func domainBlocks(domain catalog.Domain) []block {
	blocks := make([]block, 0, 2)

	if terms := termList(domain.UbiquitousLanguage); terms != "" {
		blocks = append(blocks, blockHTML("Ubiquitous language", terms))
	}

	if links := attachmentLinks(domain.Attachments); links != "" {
		blocks = append(blocks, blockHTML("Attachments", links))
	}

	return blocks
}

// domainRefs builds the relationship boxes of a domain section.
func domainRefs(domain catalog.Domain, idx *refIndex) []block {
	refs := make([]block, 0, 7)
	refs = appendRefs(refs, "Services", idx.messagesOf(kindService, idStrings(domain.Services)))
	refs = appendRefs(refs, "Sends", idx.anyOf(refIDs(domain.Sends)))
	refs = appendRefs(refs, "Receives", idx.anyOf(refIDs(domain.Receives)))
	refs = appendRefs(refs, "Sub-domains", idx.messagesOf(kindDomain, idStrings(domain.SubDomains)))
	refs = appendRefs(refs, "Entities", idx.messagesOf(kindEntity, domain.Entities))
	refs = appendRefs(refs, "Flows", idx.messagesOf(kindFlow, idStrings(domain.Flows)))
	refs = appendRefs(refs, "Data products", idx.messagesOf(kindDataProduct, idStrings(domain.DataProducts)))

	return refs
}

// termList renders a domain's ubiquitous language dictionary.
func termList(terms []catalog.UbiquitousLanguageTerm) markup {
	if len(terms) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		for _, term := range terms {
			w.open("div", htmlClass("attr"))
			w.open("div", htmlClass("attr-head"))
			w.element("span", string(term.Name), htmlClass("attr-name"))
			w.close("div")

			if term.Description != "" {
				w.element("p", term.Description, htmlClass("attr-desc"))
			}

			w.close("div")
		}
	})
}

// attachmentLinks renders external attachments as links with descriptions.
func attachmentLinks(items []catalog.Attachment) markup {
	if len(items) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("ul", htmlClass("attachments"))

		for _, item := range items {
			w.open("li")
			w.raw(externalLink(string(item.URL), defaultIfEmpty(string(item.Title), string(item.URL))))

			if item.Description != "" {
				w.element("p", string(item.Description), htmlClass("attr-desc"))
			}

			w.close("li")
		}

		w.close("ul")
	})
}
