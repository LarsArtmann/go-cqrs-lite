package architecture

import (
	"maps"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// selKey identifies one resource for kept-set bookkeeping.
type selKey struct {
	kind resourceKind
	id   string
}

// Selector addresses exactly one resource of the catalog by kind and id
// (ADR-0160). Kind names are the spellings [adrKind] accepts: "domain",
// "service", "command", "event", "query", "channel", "container" (data
// store), "flow", "entity", "data-product", "agent", "decision" (ADR),
// "team", "person" (user) and "doc".
type Selector struct {
	Kind string
	ID   string
}

// Selection is the kept-set policy of an export. The zero value keeps every
// resource, which is the whole-catalog reference of ADR-0159.
//
// Resolution rules, pinned by tests (ADR-0160):
//   - NeverExporting beats everything: a never-exported resource is absent
//     even when it is inside an included domain or explicitly included.
//   - Excluding beats including: an excluded resource drops even inside an
//     included container. Excludes name exactly the listed resources; they
//     do not cascade to members.
//   - Including a container brings its natural closure: a domain brings its
//     services and those services' messages; a service brings its messages.
type Selection struct {
	include     []Selector
	exclude     []Selector
	neverExport []Selector
}

// Including adds selectors to the include set. An empty include set keeps
// the whole catalog (minus excludes and never-exports).
func (s *Selection) Including(selectors ...Selector) *Selection {
	s.include = append(s.include, selectors...)

	return s
}

// Excluding adds selectors to the exclude set; excludes always win.
func (s *Selection) Excluding(selectors ...Selector) *Selection {
	s.exclude = append(s.exclude, selectors...)

	return s
}

// NeverExporting adds selectors that are absent from any export, even when
// explicitly included.
func (s *Selection) NeverExporting(selectors ...Selector) *Selection {
	s.neverExport = append(s.neverExport, selectors...)

	return s
}

// empty reports whether no rule restricts the export, so the whole catalog
// is kept and every render path keeps today's exact behaviour.
func (s *Selection) empty() bool {
	return s == nil || (len(s.include) == 0 && len(s.exclude) == 0 && len(s.neverExport) == 0)
}

// including returns a copy with the selectors added; nil-safe so options can
// be applied in any order without pre-seeding the exporter.
func (s *Selection) including(selectors ...Selector) *Selection {
	out := s.copy()
	out.include = append(out.include, selectors...)

	return out
}

// excluding returns a copy with the selectors added; nil-safe.
func (s *Selection) excluding(selectors ...Selector) *Selection {
	out := s.copy()
	out.exclude = append(out.exclude, selectors...)

	return out
}

// neverExporting returns a copy with the selectors added; nil-safe.
func (s *Selection) neverExporting(selectors ...Selector) *Selection {
	out := s.copy()
	out.neverExport = append(out.neverExport, selectors...)

	return out
}

// copy returns a fresh selection carrying s's rules (nil-safe).
func (s *Selection) copy() *Selection {
	if s == nil {
		return &Selection{include: nil, exclude: nil, neverExport: nil}
	}

	return &Selection{
		include:     slices.Clone(s.include),
		exclude:     slices.Clone(s.exclude),
		neverExport: slices.Clone(s.neverExport),
	}
}

// keptSet computes the resources an export may render. A nil kept result
// means "no selection": every resource is kept. The banned result lists the
// resources exclude and neverExport redact; they stay hidden in every
// references mode, even [ReferencesNameOnly].
//
// The closure of a container selector is resolved against the catalog:
// domain → its services → those services' messages. Channels, stores,
// entities, ADRs and the rest are pulled in only by their own selectors —
// they are infrastructure a consumer names deliberately. With
// [ReferencesInclude], one level of the resources a kept resource points at
// joins the set as well (a service brings its stores, entities and message
// channels), never overriding a ban.
func (s *Selection) keptSet(
	cat *catalog.Catalog,
	references ReferencesMode,
) (map[selKey]bool, map[selKey]bool) {
	if s.empty() {
		return nil, nil
	}

	kept := make(map[selKey]bool)
	banned := make(map[selKey]bool)
	resolve := newSelectorResolver(cat)

	ban := func(selectors []Selector) {
		for _, sel := range selectors {
			if kind, ok := adrKind(sel.Kind); ok {
				banned[selKey{kind: kind, id: sel.ID}] = true
			}
		}
	}

	ban(s.neverExport)
	ban(s.exclude)

	if len(s.include) == 0 {
		// No include set: keep everything the catalog has except bans.
		for _, key := range resolve.allKeys() {
			if !banned[key] {
				kept[key] = true
			}
		}

		return kept, banned
	}

	var addWithClosure func(sel Selector)

	addWithClosure = func(sel Selector) {
		kind, ok := adrKind(sel.Kind)
		if !ok {
			return
		}

		key := selKey{kind: kind, id: sel.ID}
		if kept[key] || banned[key] {
			return
		}

		if !resolve.exists(key) {
			return
		}

		kept[key] = true

		for _, member := range resolve.closure(key) {
			addWithClosure(member)
		}
	}

	for _, sel := range s.include {
		addWithClosure(sel)
	}

	if references == ReferencesInclude {
		level := slices.Collect(maps.Keys(kept))

		for _, key := range level {
			for _, edge := range resolve.edges(key) {
				addWithClosure(edge)
			}
		}
	}

	return kept, banned
}

// selectorResolver answers existence and closure questions against one
// catalog so the kept-set rules stay free of model walking.
type selectorResolver struct {
	cat *catalog.Catalog
}

func newSelectorResolver(cat *catalog.Catalog) selectorResolver {
	return selectorResolver{cat: cat}
}

// exists reports whether the catalog carries the addressed resource.
func (r selectorResolver) exists(key selKey) bool {
	switch key.kind {
	case kindOverview:
		return false
	case kindDomain:
		return r.find(
			len(r.cat.Domains),
			func(i int) bool { return r.cat.Domains[i].ID == catalog.DomainID(key.id) },
		)
	case kindService:
		return r.find(
			len(r.cat.Services),
			func(i int) bool { return r.cat.Services[i].ID == catalog.ServiceID(key.id) },
		)
	case kindCommand:
		return r.messageExists(kindCommand, key.id)
	case kindEvent:
		return r.messageExists(kindEvent, key.id)
	case kindQuery:
		return r.messageExists(kindQuery, key.id)
	case kindChannel:
		return r.find(
			len(r.cat.Channels),
			func(i int) bool { return r.cat.Channels[i].ID == catalog.ChannelID(key.id) },
		)
	case kindDataStore:
		return r.find(
			len(r.cat.DataStores),
			func(i int) bool { return r.cat.DataStores[i].ID == catalog.DataStoreID(key.id) },
		)
	case kindFlow:
		return r.find(
			len(r.cat.Flows),
			func(i int) bool { return r.cat.Flows[i].ID == catalog.FlowID(key.id) },
		)
	case kindEntity:
		return r.find(
			len(r.cat.Entities),
			func(i int) bool { return r.cat.Entities[i].ID == catalog.EntityID(key.id) },
		)
	case kindDataProduct:
		return r.find(
			len(r.cat.DataProducts),
			func(i int) bool { return r.cat.DataProducts[i].ID == catalog.DataProductID(key.id) },
		)
	case kindAgent:
		return r.find(
			len(r.cat.Agents),
			func(i int) bool { return r.cat.Agents[i].ID == catalog.AgentID(key.id) },
		)
	case kindADR:
		return r.find(
			len(r.cat.ADRs),
			func(i int) bool { return r.cat.ADRs[i].ID == catalog.ADRID(key.id) },
		)
	case kindTeam:
		return r.find(
			len(r.cat.Teams),
			func(i int) bool { return r.cat.Teams[i].ID == catalog.TeamID(key.id) },
		)
	case kindUser:
		return r.find(
			len(r.cat.Users),
			func(i int) bool { return r.cat.Users[i].ID == catalog.UserID(key.id) },
		)
	case kindDoc:
		return r.find(
			len(r.cat.CustomDocs),
			func(i int) bool { return r.cat.CustomDocs[i].ID == catalog.CustomDocID(key.id) },
		)
	default:
		return false
	}
}

// closure returns the selectors a kept container naturally brings with it.
func (r selectorResolver) closure(key selKey) []Selector {
	switch key.kind {
	case kindDomain:
		var members []Selector

		for _, domain := range r.cat.Domains {
			if string(domain.ID) != key.id {
				continue
			}

			for _, id := range domain.Services {
				members = append(members, Selector{Kind: "service", ID: string(id)})
			}
		}

		return members
	case kindService:
		var members []Selector

		for _, service := range r.cat.Services {
			if string(service.ID) != key.id {
				continue
			}

			for _, msg := range service.Commands {
				members = append(members, Selector{Kind: "command", ID: string(msg.ID)})
			}

			for _, msg := range service.Events {
				members = append(members, Selector{Kind: "event", ID: string(msg.ID)})
			}

			for _, msg := range service.Queries {
				members = append(members, Selector{Kind: "query", ID: string(msg.ID)})
			}
		}

		return members
	case kindOverview, kindCommand, kindEvent, kindQuery, kindChannel, kindDataStore,
		kindFlow, kindEntity, kindDataProduct, kindAgent, kindADR, kindTeam,
		kindUser, kindDoc:
		return nil
	default:
		return nil
	}
}

// selectorSpelling maps an internal kind to the selector spelling adrKind
// accepts (ADR-0160). Two kinds spell differently: data stores are
// "container" and ADRs are "decision"; the overview is not selectable.
func selectorSpelling(kind resourceKind) string {
	switch kind {
	case kindDataStore:
		return "container"
	case kindADR:
		return "decision"
	case kindOverview:
		return ""
	case kindDomain, kindService, kindCommand, kindEvent, kindQuery, kindChannel,
		kindFlow, kindEntity, kindDataProduct, kindAgent, kindTeam, kindUser, kindDoc:
		return string(kind)
	default:
		return string(kind)
	}
}

// edges returns the resources one level outside a kept resource that
// [ReferencesInclude] pulls into the export. Closures (domain → services,
// service → messages) are handled by [selectorResolver.closure]; edges cover
// the references the rendered sections link to: stores, entities, channels,
// a channel's messages and an ADR's appliesTo targets.
func (r selectorResolver) edges(key selKey) []Selector {
	var out []Selector

	push := func(kind resourceKind, id string) {
		out = append(out, Selector{Kind: selectorSpelling(kind), ID: id})
	}

	switch key.kind {
	case kindService:
		r.serviceEdges(key.id, push)
	case kindCommand, kindEvent, kindQuery:
		for _, channel := range r.messageChannels(key.kind, key.id) {
			push(kindChannel, channel)
		}
	case kindChannel:
		r.channelEdges(key.id, push)
	case kindFlow:
		r.flowEdges(key.id, push)
	case kindADR:
		r.adrEdges(key.id, push)
	case kindOverview, kindDomain, kindEntity, kindDataStore, kindDataProduct,
		kindAgent, kindTeam, kindUser, kindDoc:
		return nil
	default:
		return nil
	}

	return out
}

// serviceEdges pushes the stores, entities and message channels one service
// declares.
func (r selectorResolver) serviceEdges(id string, push func(resourceKind, string)) {
	for _, service := range r.cat.Services {
		if string(service.ID) != id {
			continue
		}

		for _, store := range service.ReadsFrom {
			push(kindDataStore, string(store))
		}

		for _, store := range service.WritesTo {
			push(kindDataStore, string(store))
		}

		for _, entity := range service.Entities {
			push(kindEntity, entity)
		}

		for _, msg := range slices.Concat(
			messagesOf(service, kindCommand),
			messagesOf(service, kindEvent),
			messagesOf(service, kindQuery),
		) {
			for _, channel := range msg.Channels {
				push(kindChannel, string(channel))
			}
		}
	}
}

// channelEdges pushes the messages one channel carries.
func (r selectorResolver) channelEdges(id string, push func(resourceKind, string)) {
	for _, channel := range r.cat.Channels {
		if string(channel.ID) != id {
			continue
		}

		for _, msg := range channel.Messages {
			if kind, ok := r.messageKind(string(msg)); ok {
				push(kind, string(msg))
			}
		}
	}
}

// flowEdges pushes the services and messages one flow's steps involve.
func (r selectorResolver) flowEdges(id string, push func(resourceKind, string)) {
	for _, flow := range r.cat.Flows {
		if string(flow.ID) != id {
			continue
		}

		for _, step := range flow.Steps {
			if step.Service != nil {
				push(kindService, string(step.Service.ID))
			}

			if step.Message != nil {
				if kind, ok := r.messageKind(string(step.Message.ID)); ok {
					push(kind, string(step.Message.ID))
				}
			}
		}
	}
}

// adrEdges pushes the resources one ADR's appliesTo targets name.
func (r selectorResolver) adrEdges(id string, push func(resourceKind, string)) {
	for _, adr := range r.cat.ADRs {
		if string(adr.ID) != id {
			continue
		}

		for _, target := range adr.AppliesTo {
			if kind, ok := adrKind(string(target.Type)); ok {
				push(kind, target.ID)
			}
		}
	}
}

// messageChannels returns the channel ids one message declares.
func (r selectorResolver) messageChannels(kind resourceKind, id string) []string {
	for _, service := range r.cat.Services {
		for _, msg := range messagesOf(service, kind) {
			if string(msg.ID) == id {
				return idStrings(msg.Channels)
			}
		}
	}

	return nil
}

// messageKind resolves a kindless message id to its kind by scanning the
// services that declare it.
func (r selectorResolver) messageKind(id string) (resourceKind, bool) {
	for _, kind := range []resourceKind{kindCommand, kindEvent, kindQuery} {
		if r.messageExists(kind, id) {
			return kind, true
		}
	}

	return "", false
}

// allKeys lists every resource in the catalog in registration order.
func (r selectorResolver) allKeys() []selKey {
	keys := make([]selKey, 0)

	for _, domain := range r.cat.Domains {
		keys = append(keys, selKey{kindDomain, string(domain.ID)})
	}

	for _, service := range r.cat.Services {
		keys = append(keys, selKey{kindService, string(service.ID)})

		for _, msg := range service.Commands {
			keys = append(keys, selKey{kindCommand, string(msg.ID)})
		}

		for _, msg := range service.Events {
			keys = append(keys, selKey{kindEvent, string(msg.ID)})
		}

		for _, msg := range service.Queries {
			keys = append(keys, selKey{kindQuery, string(msg.ID)})
		}
	}

	for _, channel := range r.cat.Channels {
		keys = append(keys, selKey{kindChannel, string(channel.ID)})
	}

	for _, store := range r.cat.DataStores {
		keys = append(keys, selKey{kindDataStore, string(store.ID)})
	}

	for _, flow := range r.cat.Flows {
		keys = append(keys, selKey{kindFlow, string(flow.ID)})
	}

	for _, entity := range r.cat.Entities {
		keys = append(keys, selKey{kindEntity, string(entity.ID)})
	}

	for _, product := range r.cat.DataProducts {
		keys = append(keys, selKey{kindDataProduct, string(product.ID)})
	}

	for _, agent := range r.cat.Agents {
		keys = append(keys, selKey{kindAgent, string(agent.ID)})
	}

	for _, adr := range r.cat.ADRs {
		keys = append(keys, selKey{kindADR, string(adr.ID)})
	}

	for _, team := range r.cat.Teams {
		keys = append(keys, selKey{kindTeam, string(team.ID)})
	}

	for _, user := range r.cat.Users {
		keys = append(keys, selKey{kindUser, string(user.ID)})
	}

	for _, doc := range r.cat.CustomDocs {
		keys = append(keys, selKey{kindDoc, string(doc.ID)})
	}

	return keys
}

// messageExists reports whether any service declares the message kind+id.
func (r selectorResolver) messageExists(kind resourceKind, id string) bool {
	for _, service := range r.cat.Services {
		for _, msg := range messagesOf(service, kind) {
			if string(msg.ID) == id {
				return true
			}
		}
	}

	return false
}

// find is a tiny Contains over an index predicate, keeping the switch above
// one line per kind.
func (r selectorResolver) find(n int, match func(int) bool) bool {
	for i := range n {
		if match(i) {
			return true
		}
	}

	return false
}
