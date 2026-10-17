package architecture

import (
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// channelSection builds the section for one channel.
func channelSection(channel catalog.Channel, idx *refIndex) section {
	sec := newSection(
		kindChannel,
		string(channel.ID),
		string(channel.Name),
	)

	sec.version = string(channel.Version)
	sec.summary = string(channel.Summary)

	sec.badges = channel.Badges
	sec.meta = metaLine(
		[]markup{
			ownerMeta(idx, channel.Owners),
			textMeta(string(channel.Address)),
			textMeta(string(channel.DeliveryGuarantee)),
		},
	)

	sec.left = channelBlocks(channel)
	sec.right = channelRefs(channel, idx)

	return sec
}

// channelBlocks builds the left-column blocks of a channel section.
func channelBlocks(channel catalog.Channel) []block {
	blocks := make([]block, 0)

	if params := parameterTable(channel.Parameters); params != "" {
		blocks = append(blocks, blockHTML("Parameters", params))
	}

	if routes := routeList(channel.Routes); routes != "" {
		blocks = append(blocks, blockHTML("Routes", routes))
	}

	return blocks
}

// channelRefs builds the relationship boxes of a channel section.
func channelRefs(channel catalog.Channel, idx *refIndex) []block {
	refs := make([]block, 0)
	refs = appendRefs(refs, "Messages", idx.anyOf(idStrings(channel.Messages)))
	refs = appendRefs(refs, "Owners", idx.owners(channel.Owners))

	if len(channel.Protocols) > 0 {
		refs = append(
			refs,
			blockCounted(
				"Protocols",
				len(channel.Protocols),
				plainRefList(idStrings(channel.Protocols)),
			),
		)
	}

	return refs
}

// parameterTable renders a channel's address parameters.
func parameterTable(params map[string]catalog.ChannelParam) markup {
	if len(params) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md-table"))
		w.open("table")
		w.open("thead")
		w.open("tr")

		for _, header := range []string{"Parameter", headerDescription, "Default"} {
			w.element("th", header)
		}

		w.close("tr")
		w.close("thead")
		w.open("tbody")

		for _, name := range sortedKeysOf(params) {
			param := params[name]

			w.open("tr")
			w.element("td", name, htmlClass("mono"))
			w.element("td", string(param.Description))
			w.element("td", param.Default, htmlClass("mono"))
			w.close("tr")
		}

		w.close("tbody")
		w.close("table")
		w.close("div")
	})
}

// routeList renders a channel's routing rules.
func routeList(routes []catalog.ChannelRoute) markup {
	if len(routes) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		for _, route := range routes {
			w.open("div", htmlClass("route"))
			w.element("span", string(route.ID), htmlClass("mono"))
			w.element("span", "→", htmlClass("route-arrow"))
			w.element("span", joinIDs(idStrings(route.To)))
			w.close("div")
		}
	})
}

// plainRefList renders plain values as an in-document list.
func plainRefList(values []string) markup {
	return refList(plainRefs(values))
}

// joinIDs joins identifiers for inline display.
func joinIDs(values []string) string {
	if len(values) == 0 {
		return ""
	}

	var b strings.Builder

	for i, value := range values {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(value)
	}

	return b.String()
}

// dataStoreSection builds the section for one data store.
func dataStoreSection(store catalog.DataStore, idx *refIndex, cat *catalog.Catalog) section {
	meta := []markup{
		ownerMeta(idx, store.Owners),
		textMeta(store.Technology),
	}

	if store.Authoritative {
		meta = append(meta, textMeta("System of record"))
	}

	sec := newSection(
		kindDataStore,
		string(store.ID),
		string(store.Name),
	)

	sec.version = string(store.Version)
	sec.summary = string(store.Summary)

	sec.badges = store.Badges
	sec.meta = metaLine(meta)

	sec.left = []block{blockHTML("Store", dataStoreFields(store))}
	sec.right = dataStoreRefs(store, idx, cat)

	return sec
}

// dataStoreFields renders a data store's operational attributes.
func dataStoreFields(store catalog.DataStore) markup {
	return fieldList([][2]string{
		{headerType, store.ContainerType},
		{"Technology", store.Technology},
		{"Classification", store.Classification},
		{"Retention", store.Retention},
		{"Residency", store.Residency},
		{"Access mode", store.AccessMode},
	})
}

// dataStoreRefs builds the relationship boxes of a data store section,
// including the services that read and write it.
func dataStoreRefs(store catalog.DataStore, idx *refIndex, cat *catalog.Catalog) []block {
	writers, readers := servicesUsing(cat, string(store.ID))

	refs := make([]block, 0)
	refs = appendRefs(refs, "Written by", idx.messagesOf(kindService, writers))
	refs = appendRefs(refs, "Read by", idx.messagesOf(kindService, readers))
	refs = appendRefs(refs, "Owners", idx.owners(store.Owners))

	return refs
}

// servicesUsing splits the catalog's services into the ones that write to a
// data store and the ones that read from it.
func servicesUsing(cat *catalog.Catalog, storeID string) ([]string, []string) {
	writers := make([]string, 0, len(cat.Services))
	readers := make([]string, 0, len(cat.Services))

	for _, service := range cat.Services {
		if containsID(idStrings(service.WritesTo), storeID) {
			writers = append(writers, string(service.ID))
		}

		if containsID(idStrings(service.ReadsFrom), storeID) {
			readers = append(readers, string(service.ID))
		}
	}

	return writers, readers
}

// containsID reports whether values holds id.
func containsID(values []string, id string) bool {
	return slices.Contains(values, id)
}
