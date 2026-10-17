package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// collect turns a catalog into a renderable document: it indexes every
// resource first (so links resolve in both directions) and then builds one
// section per resource, in the order the sidebar lists them.
func collect(cat *catalog.Catalog, cfg config) *document {
	enriched := cat.DeriveProducersConsumers()
	idx := newRefIndex()

	register(enriched, idx)

	b := newDocBuilder(cfg.title, cfg.command, cfg.generatedAt)
	b.sections = append(b.sections, overviewSection(enriched, cfg, idx))
	b.navAt(kindOverview, navEntry{anchor: "overview", tag: "DOC", class: "doc", name: "Introduction"})

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
	collectUsers(b, enriched, idx)
	collectDocs(b, enriched, idx)
	collectCounts(b, enriched)

	return &document{
		title:       cfg.title,
		description: cfg.description,
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
		anchor: sec.anchor,
		tag:    info.tag,
		class:  info.class,
		name:   sec.name,
		alt:    slug(sec.name),
	}
}

// addResource appends a section and its sidebar entry, and records the search
// hit in one place so the three never drift apart.
func addResource(b *docBuilder, sec section, domain string) {
	b.addSection(sec, navEntryFor(sec))
	b.search = append(b.search, searchEntry{
		Group:   kindInfos()[sec.kind].group,
		Key:     kindInfos()[sec.kind].tag,
		Class:   kindInfos()[sec.kind].class,
		Title:   sec.name,
		Alt:     slug(sec.name),
		Summary: sec.summary,
		Domain:  domain,
		Href:    sec.anchor,
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
		addResource(b, serviceSection(service, idx), domainOfService(cat, service.ID))
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
		addResource(b, dataStoreSection(store, idx), "")
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
func collectUsers(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, user := range cat.Users {
		addResource(b, userSection(user, idx), "")
	}
}

// collectDocs adds one section per custom documentation page.
func collectDocs(b *docBuilder, cat *catalog.Catalog, idx *refIndex) {
	for _, doc := range cat.CustomDocs {
		addResource(b, docSection(doc, idx), "")
	}
}

// collectCounts records the summary numbers shown on the overview.
func collectCounts(b *docBuilder, cat *catalog.Catalog) {
	counts := countMessages(cat)

	b.addCount("Domains", len(cat.Domains))
	b.addCount("Services", len(cat.Services))
	b.addCount("Commands", counts[kindCommand])
	b.addCount("Events", counts[kindEvent])
	b.addCount("Queries", counts[kindQuery])
	b.addCount("Channels", len(cat.Channels))
	b.addCount("Data stores", len(cat.DataStores))
	b.addCount("Flows", len(cat.Flows))
	b.addCount("Entities", len(cat.Entities))
	b.addCount("Data products", len(cat.DataProducts))
	b.addCount("Agents", len(cat.Agents))
	b.addCount("Decisions", len(cat.ADRs))
	b.addCount("Teams", len(cat.Teams))
	b.addCount("People", len(cat.Users))
}

// countMessages counts unique messages per kind.
func countMessages(cat *catalog.Catalog) map[resourceKind]int {
	counts := make(map[resourceKind]int)

	for _, kind := range []resourceKind{kindCommand, kindEvent, kindQuery} {
		seen := make(map[string]struct{})

		for _, service := range cat.Services {
			for _, msg := range messagesOf(service, kind) {
				seen[string(msg.ID)] = struct{}{}
			}
		}

		counts[kind] = len(seen)
	}

	return counts
}

// messagesOf returns a service's messages of one kind.
func messagesOf(service catalog.Service, kind resourceKind) []catalog.Message {
	switch kind {
	case kindCommand:
		return service.Commands
	case kindEvent:
		return service.Events
	case kindQuery:
		return service.Queries
	default:
		return nil
	}
}

// domainOfService returns the name of the domain owning a service, if any.
func domainOfService(cat *catalog.Catalog, id catalog.ServiceID) string {
	for _, domain := range cat.Domains {
		for _, member := range domain.Services {
			if member == id {
				return string(domain.Name)
			}
		}
	}

	return ""
}
