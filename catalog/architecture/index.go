package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// refIndex resolves catalog identifiers to in-document anchors, names and
// sidebar tags. Every link in the reference goes through it, so a resource
// that is present gets a link and one that is missing degrades to plain text.
// Under a selection (ADR-0160), only kept resources are registered and the
// resolvers skip references to anything outside the kept set, so a redacted
// export leaks neither links nor names.
type refIndex struct {
	byKind map[resourceKind]map[string]resourceRef
	byID   map[string]resourceRef
	// channelsByMessage inverts Channel.Messages, so a message section can
	// list the channels that carry it without rescanning the catalog.
	channelsByMessage map[string][]string
	// kept is nil when no selection is active: everything is kept.
	kept map[selKey]bool
	// banned lists the resources exclude and neverExport redact; they stay
	// hidden in every references mode. bannedIDs flattens it for the
	// resolvers that see identifiers without a kind.
	banned    map[selKey]bool
	bannedIDs map[string]bool
	// references selects how out-of-scope references degrade (ADR-0160).
	references ReferencesMode
	// hide records which reference parts a redacted export suppresses.
	hide visibility
	// names records every non-redacted resource's display name, including
	// resources outside the kept set, so name-only mode can speak of them.
	names     map[selKey]string
	namesByID map[string]string
}

// newRefIndex creates an empty index.
func newRefIndex() *refIndex {
	return &refIndex{
		byKind:            make(map[resourceKind]map[string]resourceRef),
		byID:              make(map[string]resourceRef),
		channelsByMessage: make(map[string][]string),
		kept:              nil,
		banned:            nil,
		bannedIDs:         make(map[string]bool),
		references:        ReferencesOmit,
		names:             make(map[selKey]string),
		namesByID:         make(map[string]string),
	}
}

// isKept reports whether a resource survives the export selection. A nil
// kept map means no selection: everything is kept.
func (x *refIndex) isKept(kind resourceKind, id string) bool {
	return x.kept == nil || x.kept[selKey{kind: kind, id: id}]
}

// add registers a resource and returns the reference other sections link to.
// Resources outside the kept set are not registered, so every resolver below
// treats them as absent from the document.
func (x *refIndex) add(kind resourceKind, id, name string) {
	key := selKey{kind: kind, id: id}

	if !x.banned[key] {
		x.names[key] = name

		if _, exists := x.namesByID[id]; !exists {
			x.namesByID[id] = name
		}
	}

	if !x.isKept(kind, id) {
		return
	}

	info := kindInfos()[kind]
	ref := resourceRef{
		anchor: info.anchor(id),
		name:   name,
		tag:    info.tag,
		class:  info.class,
		note:   "",
	}

	if x.byKind[kind] == nil {
		x.byKind[kind] = make(map[string]resourceRef)
	}

	x.byKind[kind][id] = ref

	if _, exists := x.byID[id]; !exists {
		x.byID[id] = ref
	}
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

// owners renders owner identifiers as resolvable references. Under a
// selection, owners outside the kept set (or unknown) are dropped entirely:
// a redacted export must not carry an excluded team or person as text.
func (x *refIndex) owners(ids []string) []resourceRef {
	if x.hide.owners {
		return nil
	}

	refs := make([]resourceRef, 0, len(ids))

	for _, id := range ids {
		ref := x.owner(id)

		if ref.anchor != "" {
			refs = append(refs, ref)

			continue
		}

		teamKey, userKey := selKey{kind: kindTeam, id: id}, selKey{kind: kindUser, id: id}

		if x.banned[teamKey] || x.banned[userKey] {
			continue
		}

		if x.references == ReferencesNameOnly {
			name := x.names[teamKey]

			if name == "" {
				name = x.names[userKey]
			}

			if name != "" {
				refs = append(refs, plainRef(name))

				continue
			}

			if x.kept != nil {
				continue
			}
		}

		if x.kept == nil {
			refs = append(refs, plainRef(id))
		}
	}

	return refs
}

// anyOf resolves identifiers whose kind the model does not record, such as the
// messages a domain sends or the messages listed on a channel. Under a
// selection, unresolvable identifiers are skipped instead of degrading, so
// the export carries no trace of resources it omitted.
func (x *refIndex) anyOf(ids []string) []resourceRef {
	refs := make([]resourceRef, 0, len(ids))

	for _, id := range ids {
		//art-dupl:accept accidental shape match with cmd/cqrs-lint/discordsync_fixture_test.go accumulate loop: shared continue-ladder shape only, not meaning (ref resolution vs finding accumulation)
		ref := x.any(id)

		if ref.anchor == "" {
			var ok bool

			if ref, ok = x.degradeAny(id); !ok {
				continue
			}
		}

		refs = append(refs, ref)
	}

	return refs
}

// messagesOf narrows a message id to a reference of a specific kind. Under
// a selection, unresolvable identifiers are skipped (see anyOf).
func (x *refIndex) messagesOf(kind resourceKind, ids []string) []resourceRef {
	refs := make([]resourceRef, 0, len(ids))

	for _, id := range ids {
		if ref := x.get(kind, id); ref.anchor != "" {
			refs = append(refs, ref)

			continue
		}

		ref := x.any(id)

		if ref.anchor == "" {
			var ok bool

			if ref, ok = x.degradeAny(id); !ok {
				continue
			}
		}

		refs = append(refs, ref)
	}

	return refs
}

// degrade resolves a typed identifier the index cannot link. Redacted
// resources (exclude/neverExport) drop in every mode; name-only mode
// renders the display name as plain text; without a selection the
// identifier degrades to plain text as it always has.
func (x *refIndex) degrade(kind resourceKind, id string) (resourceRef, bool) {
	key := selKey{kind: kind, id: id}

	if x.banned[key] {
		return droppedRef(), false
	}

	if x.references == ReferencesNameOnly {
		if name := x.names[key]; name != "" {
			return plainRef(name), true
		}

		if x.kept != nil {
			return droppedRef(), false
		}
	}

	if x.kept == nil {
		return plainRef(id), true
	}

	return droppedRef(), false
}

// degradeAny is degrade for identifiers the model stores without a kind.
// The no-selection fallback appends a zero reference, which writeRef skips.
func (x *refIndex) degradeAny(id string) (resourceRef, bool) {
	if x.bannedIDs[id] {
		return droppedRef(), false
	}

	if x.references == ReferencesNameOnly {
		if name := x.namesByID[id]; name != "" {
			return plainRef(name), true
		}

		if x.kept != nil {
			return droppedRef(), false
		}
	}

	if x.kept == nil {
		return droppedRef(), true
	}

	return droppedRef(), false
}

// channelsFor returns the channels a message travels on, merging the
// message's own channel list with the reverse index built from the channels.
func (x *refIndex) channelsFor(msg catalog.Message) []resourceRef {
	ids := idStrings(msg.Channels)

	for _, id := range x.channelsByMessage[string(msg.ID)] {
		if !containsID(ids, id) {
			ids = append(ids, id)
		}
	}

	return x.messagesOf(kindChannel, ids)
}

// link resolves a "[[kind|id]]" prose reference to an in-document link. It
// understands the same kind names ADR targets use, so one spelling works in
// decisions, docs and export selectors alike.
func (x *refIndex) link(kind, id string) resourceRef {
	resolved, ok := adrKind(kind)
	if !ok {
		if x.kept != nil {
			return droppedRef()
		}

		return plainRef(id)
	}

	if ref := x.get(resolved, id); ref.anchor != "" {
		return ref
	}

	if ref, keep := x.degrade(resolved, id); keep {
		return ref
	}

	return droppedRef()
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

		for _, id := range channel.Messages {
			idx.channelsByMessage[string(id)] = append(
				idx.channelsByMessage[string(id)],
				string(channel.ID),
			)
		}
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
