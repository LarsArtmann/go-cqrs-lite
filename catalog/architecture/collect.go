package architecture

import (
	"slices"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// collect turns a catalog into a renderable document: it indexes every
// resource first (so links resolve in both directions) and then builds one
// section per resource, in the order the sidebar lists them. With a
// selection (ADR-0160), only kept resources are indexed, counted and
// rendered, so no link, sidebar entry or search hit points outside the set.
func collect(cat *catalog.Catalog, cfg config) *document {
	enriched := cat.DeriveProducersConsumers()
	plan := cfg.sel.plan(enriched, cfg.references)
	idx := newRefIndex()
	idx.kept = plan.kept
	idx.references = cfg.references
	idx.hide = cfg.hide
	idx.banned = plan.banned

	for key := range plan.banned {
		idx.bannedIDs[key.id] = true
	}

	register(enriched, idx)

	b := newDocBuilder(cfg.title, cfg.command, cfg.generatedAt)
	b.kept = plan.kept
	collectCounts(b, enriched)
	b.sections = append(b.sections, overviewSection(cfg, b.counts))
	b.navAt(kindOverview, navEntry{
		anchor: "overview",
		tag:    "DOC",
		class:  "doc",
		name:   "Introduction",
		alt:    "introduction",
	})

	collectDomains(b, enriched, idx)
	collectServices(b, enriched, idx)
	collectMessages(b, enriched, idx)
	collectChannels(b, enriched, idx)
	collectDataStores(b, enriched, idx)
	collectFlows(b, enriched, idx)
	collectEntities(b, enriched, idx)
	collectDataProducts(b, enriched, idx)
	collectAgents(b, enriched, idx)
	collectADRs(b, enriched, idx)
	collectTeams(b, enriched, idx)
	collectUsers(b, enriched)
	collectDocs(b, enriched, idx)

	return &document{
		title:       cfg.title,
		description: cfg.description,
		logo:        safeImageSrc(cfg.logo),
		generated:   cfg.generatedAt,
		command:     cfg.command,
		counts:      b.counts,
		nav:         b.nav,
		sections:    b.sections,
		search:      b.search,
	}
}

// navEntryFor derives a resource's sidebar entry from its finished section.
func navEntryFor(sec section) navEntry {
	info := kindInfos()[sec.kind]

	return navEntry{
		anchor: sec.anchor(),
		tag:    info.tag,
		class:  info.class,
		name:   sec.name,
		alt:    slug(sec.name),
	}
}

// addResource appends a section and its sidebar entry, and records the search
// hit in one place so the three never drift apart. Resources outside the
// export selection (ADR-0160) are skipped entirely.
func addResource(b *docBuilder, sec section, domain string) {
	if !b.isKept(sec.kind, sec.id) {
		return
	}

	b.addSection(sec, navEntryFor(sec))
	b.search = append(b.search, searchEntry{
		Group:   kindInfos()[sec.kind].group,
		Key:     kindInfos()[sec.kind].tag,
		Class:   kindInfos()[sec.kind].class,
		Title:   sec.name,
		Alt:     slug(sec.name),
		Summary: sec.summary,
		Domain:  domain,
		Href:    sec.anchor(),
	})
}

// collectDomains adds one section per domain.
func collectDomains(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, domain := range cat.Domains {
		addResource(b, domainSection(domain, idx), "")
	}
}

// collectServices adds one section per service.
func collectServices(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, service := range cat.Services {
		addResource(b, serviceSection(service, idx), domainOfService(cat, service.ID, b))
	}
}

// collectMessages adds one section per unique message, in kind order.
func collectMessages(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, kind := range []resourceKind{kindCommand, kindEvent, kindQuery} {
		seen := make(map[string]struct{})

		for _, service := range cat.Services {
			for _, msg := range messagesOf(service, kind) {
				if _, dup := seen[string(msg.ID)]; dup {
					continue
				}

				seen[string(msg.ID)] = struct{}{}
				addResource(b, messageSection(msg, kind, idx), service.Name.String())
			}
		}
	}
}

// collectChannels adds one section per channel.
func collectChannels(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, channel := range cat.Channels {
		addResource(b, channelSection(channel, idx), "")
	}
}

// collectDataStores adds one section per data store.
func collectDataStores(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, store := range cat.DataStores {
		addResource(b, dataStoreSection(store, idx, cat), "")
	}
}

// collectFlows adds one section per flow.
func collectFlows(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, flow := range cat.Flows {
		addResource(b, flowSection(flow, idx), "")
	}
}

// collectEntities adds one section per entity.
func collectEntities(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, entity := range cat.Entities {
		addResource(b, entitySection(entity, idx), "")
	}
}

// collectDataProducts adds one section per data product.
func collectDataProducts(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, product := range cat.DataProducts {
		addResource(b, dataProductSection(product, idx), "")
	}
}

// collectAgents adds one section per agent.
func collectAgents(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, agent := range cat.Agents {
		addResource(b, agentSection(agent, idx), "")
	}
}

// collectADRs adds one section per architecture decision record.
func collectADRs(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, adr := range cat.ADRs {
		addResource(b, adrSection(adr, idx), "")
	}
}

// collectTeams adds one section per team.
func collectTeams(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, team := range cat.Teams {
		addResource(b, teamSection(team, idx), "")
	}
}

// collectUsers adds one section per person.
func collectUsers(b *docBuilder, cat *catalog.Catalog) {
	for _, user := range cat.Users {
		addResource(b, userSection(user), "")
	}
}

// collectDocs adds one section per custom documentation page.
func collectDocs(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	if idx.hide.docs {
		return
	}

	for _, doc := range cat.CustomDocs {
		addResource(b, docSection(doc, idx), "")
	}
}

// collectCounts records the summary numbers shown on the overview. Under a
// selection, only kept resources are counted so the overview cannot claim
// more than the document shows.
func collectCounts(b *docBuilder, cat *catalog.Catalog) {
	counted := make(map[resourceKind]int)

	for _, kind := range []resourceKind{kindCommand, kindEvent, kindQuery} {
		seen := make(map[string]struct{})

		for _, service := range cat.Services {
			for _, msg := range messagesOf(service, kind) {
				if !b.isKept(kind, string(msg.ID)) {
					continue
				}

				seen[string(msg.ID)] = struct{}{}
			}
		}

		counted[kind] = len(seen)
	}

	b.addCount("Domains", "domain", b.keptCountOr(kindDomain, len(cat.Domains)))
	b.addCount("Services", "service", b.keptCountOr(kindService, len(cat.Services)))
	b.addCount("Commands", "command", counted[kindCommand])
	b.addCount("Events", "event", counted[kindEvent])
	b.addCount("Queries", "query", counted[kindQuery])
	b.addCount("Channels", "channel", b.keptCountOr(kindChannel, len(cat.Channels)))
	b.addCount("Data stores", "data store", b.keptCountOr(kindDataStore, len(cat.DataStores)))
	b.addCount("Flows", "flow", b.keptCountOr(kindFlow, len(cat.Flows)))
	b.addCount("Entities", "entity", b.keptCountOr(kindEntity, len(cat.Entities)))
	b.addCount(
		"Data products",
		"data product",
		b.keptCountOr(kindDataProduct, len(cat.DataProducts)),
	)
	b.addCount("Agents", "agent", b.keptCountOr(kindAgent, len(cat.Agents)))
	b.addCount("Decisions", "decision", b.keptCountOr(kindADR, len(cat.ADRs)))
	b.addCount("Teams", "team", b.keptCountOr(kindTeam, len(cat.Teams)))
	b.addCount("People", "person", b.keptCountOr(kindUser, len(cat.Users)))
}

// keptCountOr returns how many resources of a kind survive the selection, or
// the catalog total when no selection is active.
func (d *docBuilder) keptCountOr(kind resourceKind, total int) int {
	if d.kept == nil {
		return total
	}

	n := 0

	for key := range d.kept {
		if key.kind == kind {
			n++
		}
	}

	return n
}

// messagesOf returns a service's messages of one kind.
func messagesOf(service catalog.Service, kind resourceKind) []catalog.Message {
	if kind == kindCommand {
		return service.Commands
	}

	if kind == kindEvent {
		return service.Events
	}

	if kind == kindQuery {
		return service.Queries
	}

	return nil
}

// domainOfService returns the name of the domain owning a service, if any.
// Under a selection that drops the owning domain, the meta text is dropped
// too: a redacted export must not carry the excluded name.
func domainOfService(cat *catalog.Catalog, id catalog.ServiceID, b *docBuilder) string {
	for _, domain := range cat.Domains {
		if slices.Contains(domain.Services, id) {
			if !b.isKept(kindDomain, string(domain.ID)) {
				return ""
			}

			return string(domain.Name)
		}
	}

	return ""
}
