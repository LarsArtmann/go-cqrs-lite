package catalog

// Demo marking: a registry flagged with MarkDemo() gets every declared
// resource stamped at Build() time, so "this is demo content" is a property
// of the catalog, not per-declaration discipline a future edit can forget.
//
// The stamp respects the downstream renderers' schemas:
//   - Badge-capable resources (services, their messages, domains, channels,
//     data stores, flows, entities, data products, agents, custom docs) get
//     the canonical demo badge, appended after any caller-declared badges.
//   - Teams and users have no badges in EventCatalog; they get the demo
//     Source (provider "demo"), the rendered provenance field — unless the
//     declaration already carries a real source, which always wins.

// demoBadgeContent identifies the canonical demo badge. Stamping is
// idempotent on this value: an explicit badge of the same content is kept
// as-is and never duplicated.
const demoBadgeContent = "Demo"

// demoBadge is the canonical marker for demo content: amber chip, dark
// text, readable in both light and dark EventCatalog themes.
func demoBadge() Badge {
	return Badge{
		Content:         demoBadgeContent,
		BackgroundColor: "#f59e0b",
		TextColor:       "#1f2937",
	}
}

// demoSource marks a team or user record as demo provenance.
func demoSource() *Source {
	return &Source{Provider: "demo"}
}

// appendDemoBadge returns a fresh slice with the demo badge appended unless
// one with the canonical content is already present. Always allocating
// keeps the stamp from aliasing registry-owned backing arrays.
func appendDemoBadge(badges []Badge) []Badge {
	for _, b := range badges {
		if b.Content == demoBadgeContent {
			return badges
		}
	}

	out := make([]Badge, 0, len(badges)+1)
	out = append(out, badges...)

	return append(out, demoBadge())
}

// markDemoCatalog stamps a built catalog copy in place; Build() hands it a
// fresh deep copy, so the registry itself never carries the stamps.
func markDemoCatalog(cat *Catalog) {
	for i := range cat.Services {
		svc := &cat.Services[i]
		svc.Badges = appendDemoBadge(svc.Badges)
		svc.Commands = markDemoMessages(svc.Commands)
		svc.Events = markDemoMessages(svc.Events)
		svc.Queries = markDemoMessages(svc.Queries)
	}

	for i := range cat.Domains {
		cat.Domains[i].Badges = appendDemoBadge(cat.Domains[i].Badges)
	}

	for i := range cat.Channels {
		cat.Channels[i].Badges = appendDemoBadge(cat.Channels[i].Badges)
	}

	for i := range cat.DataStores {
		cat.DataStores[i].Badges = appendDemoBadge(cat.DataStores[i].Badges)
	}

	for i := range cat.Flows {
		cat.Flows[i].Badges = appendDemoBadge(cat.Flows[i].Badges)
	}

	for i := range cat.Entities {
		cat.Entities[i].Badges = appendDemoBadge(cat.Entities[i].Badges)
	}

	for i := range cat.DataProducts {
		cat.DataProducts[i].Badges = appendDemoBadge(cat.DataProducts[i].Badges)
	}

	for i := range cat.Agents {
		cat.Agents[i].Badges = appendDemoBadge(cat.Agents[i].Badges)
	}

	for i := range cat.CustomDocs {
		cat.CustomDocs[i].Badges = appendDemoBadge(cat.CustomDocs[i].Badges)
	}

	for i := range cat.Teams {
		if cat.Teams[i].Source == nil {
			cat.Teams[i].Source = demoSource()
		}
	}

	for i := range cat.Users {
		if cat.Users[i].Source == nil {
			cat.Users[i].Source = demoSource()
		}
	}
}

func markDemoMessages(msgs []Message) []Message {
	for i := range msgs {
		msgs[i].Badges = appendDemoBadge(msgs[i].Badges)
	}

	return msgs
}
