package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// refIndex resolves catalog identifiers to in-document anchors, names and
// sidebar tags. Every link in the reference goes through it, so a resource
// that is present gets a link and one that is missing degrades to plain text.
type refIndex struct {
	byKind map[resourceKind]map[string]resourceRef
	byID   map[string]resourceRef
}

// newRefIndex creates an empty index.
func newRefIndex() *refIndex {
	return &refIndex{
		byKind: make(map[resourceKind]map[string]resourceRef),
		byID:   make(map[string]resourceRef),
	}
}

// add registers a resource and returns the reference other sections link to.
func (x *refIndex) add(kind resourceKind, id, name string) resourceRef {
	info := kindInfos()[kind]
	ref := resourceRef{
		anchor: info.anchor(id),
		name:   name,
		tag:    info.tag,
		class:  info.class,
	}

	if x.byKind[kind] == nil {
		x.byKind[kind] = make(map[string]resourceRef)
	}

	x.byKind[kind][id] = ref

	if _, exists := x.byID[id]; !exists {
		x.byID[id] = ref
	}

	return ref
}

// get returns the reference for a resource of a known kind.
func (x *refIndex) get(kind resourceKind, id string) resourceRef {
	return x.byKind[kind][id]
}

// any returns the reference for whichever resource owns an identifier. It is
// the fallback for identifiers the model stores without a kind, such as the
// messages listed on a channel.
func (x *refIndex) any(id string) resourceRef {
	return x.byID[id]
}

// owner resolves an owner identifier to a team or user reference.
func (x *refIndex) owner(id string) resourceRef {
	if ref := x.get(kindTeam, id); ref.anchor != "" {
		return ref
	}

	return x.get(kindUser, id)
}

// owners renders owner identifiers as resolvable references.
func (x *refIndex) owners(ids []string) []resourceRef {
	refs := make([]resourceRef, 0, len(ids))

	for _, id := range ids {
		ref := x.owner(id)
		if ref.anchor == "" {
			ref = resourceRef{name: id}
		}

		refs = append(refs, ref)
	}

	return refs
}

// anyOf resolves identifiers whose kind the model does not record, such as the
// messages a domain sends or the messages listed on a channel.
func (x *refIndex) anyOf(ids []string) []resourceRef {
	refs := make([]resourceRef, 0, len(ids))

	for _, id := range ids {
		refs = append(refs, x.any(id))
	}

	return refs
}

// messagesOf narrows a message id to a reference of a specific kind.func (x *refIndex) messagesOf(kind resourceKind, ids []string) []resourceRef {
	refs := make([]resourceRef, 0, len(ids))

	for _, id := range ids {
		if ref := x.get(kind, id); ref.anchor != "" {
			refs = append(refs, ref)

			continue
		}

		refs = append(refs, x.any(id))
	}

	return refs
}

// idStrings converts a slice of string-backed identifiers to plain strings.
func idStrings[T ~string](values []T) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

// refIDs extracts the message identifiers from catalog references.
func refIDs(refs []catalog.Ref) []string {
	out := make([]string, 0, len(refs))

	for _, ref := range refs {
		out = append(out, string(ref.ID))
	}

	return out
}

// register indexes every resource in the catalog before any section is built.
func register(cat *catalog.Catalog, idx *refIndex) {
	for _, domain := range cat.Domains {
		idx.add(kindDomain, string(domain.ID), string(domain.Name))
	}

	for _, store := range cat.DataStores {
		idx.add(kindDataStore, string(store.ID), string(store.Name))
	}

	for _, channel := range cat.Channels {
		idx.add(kindChannel, string(channel.ID), string(channel.Name))
	}

	for _, service := range cat.Services {
		idx.add(kindService, string(service.ID), string(service.Name))
	}

	registerMessages(cat, idx)

	for _, flow := range cat.Flows {
		idx.add(kindFlow, string(flow.ID), string(flow.Name))
	}

	for _, entity := range cat.Entities {
		idx.add(kindEntity, string(entity.ID), string(entity.Name))
	}

	for _, product := range cat.DataProducts {
		idx.add(kindDataProduct, string(product.ID), string(product.Name))
	}

	for _, agent := range cat.Agents {
		idx.add(kindAgent, string(agent.ID), string(agent.Name))
	}

	for _, adr := range cat.ADRs {
		idx.add(kindADR, string(adr.ID), string(adr.Name))
	}

	for _, team := range cat.Teams {
		idx.add(kindTeam, string(team.ID), string(team.Name))
	}

	for _, user := range cat.Users {
		idx.add(kindUser, string(user.ID), string(user.Name))
	}

	for _, doc := range cat.CustomDocs {
		idx.add(kindDoc, string(doc.ID), string(doc.Title))
	}
}

// registerMessages indexes events before commands and queries so that a bare
// message identifier shared across kinds resolves to the event, matching how
// channels and flows reference messages.
func registerMessages(cat *catalog.Catalog, idx *refIndex) {
	for _, order := range []struct {
		kind    resourceKind
		collect func(catalog.Service) []catalog.Message
	}{
		{kindEvent, func(s catalog.Service) []catalog.Message { return s.Events }},
		{kindCommand, func(s catalog.Service) []catalog.Message { return s.Commands }},
		{kindQuery, func(s catalog.Service) []catalog.Message { return s.Queries }},
	} {
		for _, service := range cat.Services {
			for _, msg := range order.collect(service) {
				idx.add(order.kind, string(msg.ID), string(msg.Name))
			}
		}
	}
}
