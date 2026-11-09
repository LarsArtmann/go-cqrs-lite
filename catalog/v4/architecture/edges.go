package architecture

import "slices"

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
