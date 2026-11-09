package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

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
