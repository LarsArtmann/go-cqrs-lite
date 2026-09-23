package eventcatalog

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// writeAllMessages writes every event, command, and query exactly once to
// the canonical top-level EventCatalog directories (events/, commands/,
// queries/). A message shared by several services (e.g. an event that one
// service sends and another receives) previously produced one duplicate page
// per service under services/<svc>/...; the dedupe map writes it once, with
// producers/consumers derived across all services by
// autoDeriveProducersConsumers.
func (e *Exporter) writeAllMessages(cat *catalog.Catalog) error {
	type kindMessages struct {
		kind     string
		messages []catalog.Message
	}

	written := make(map[string]struct{})
	serviceVersions := serviceVersionsOf(cat)
	channelVersions := channelVersionsOf(cat)

	for _, group := range []kindMessages{
		{kind: "commands", messages: commandsOf(cat)},
		{kind: "events", messages: eventsOf(cat)},
		{kind: "queries", messages: queriesOf(cat)},
	} {
		for _, msg := range group.messages {
			key := group.kind + "/" + string(catalog.Key(msg))
			if _, seen := written[key]; seen {
				continue
			}
			written[key] = struct{}{}

			err := e.writeMessage(group.kind, msg, serviceVersions, channelVersions)
			if err != nil {
				return errorfamily.Newf(
					errorfamily.Infrastructure,
					"catalog.exporter.9",
					"write %s %s: %v",
					group.kind,
					catalog.Key(msg),
					err,
				)
			}
		}
	}

	return nil
}

func commandsOf(cat *catalog.Catalog) []catalog.Message {
	var out []catalog.Message
	for _, svc := range cat.Services {
		out = append(out, svc.Commands...)
	}

	return out
}

func eventsOf(cat *catalog.Catalog) []catalog.Message {
	var out []catalog.Message
	for _, svc := range cat.Services {
		out = append(out, svc.Events...)
	}

	return out
}

func queriesOf(cat *catalog.Catalog) []catalog.Message {
	var out []catalog.Message
	for _, svc := range cat.Services {
		out = append(out, svc.Queries...)
	}

	return out
}

// serviceVersionsOf maps each service ID to its declared version so message
// frontmatter can emit EventCatalog reference strings ("<id>-<version>").
func serviceVersionsOf(cat *catalog.Catalog) map[catalog.ServiceID]catalog.Version {
	versions := make(map[catalog.ServiceID]catalog.Version, len(cat.Services))
	for _, svc := range cat.Services {
		versions[svc.ID] = svc.Version
	}

	return versions
}

// channelVersionsOf maps each channel ID to its declared version so message
// channel pointers can carry explicit versions.
func channelVersionsOf(cat *catalog.Catalog) map[catalog.ChannelID]catalog.Version {
	versions := make(map[catalog.ChannelID]catalog.Version, len(cat.Channels))
	for _, ch := range cat.Channels {
		versions[ch.ID] = ch.Version
	}

	return versions
}

// channelMessageIndex resolves every message in the catalog to the fully
// qualified pointer EventCatalog's channel frontmatter requires:
// {collection, name, id, version}. Bare {id, version} pointers fail schema
// validation ("messages.0.collection: Required"), and the id must carry the
// version-qualified Astro entry ID ("<messageID>-<version>") or the build
// logs "Invalid content reference" and the links never resolve.
func channelMessageIndex(cat *catalog.Catalog) map[catalog.MessageID]channelMessageFM {
	index := make(map[catalog.MessageID]channelMessageFM)

	type kindMessages struct {
		collection string
		messages   []catalog.Message
	}

	for _, group := range []kindMessages{
		{collection: "commands", messages: commandsOf(cat)},
		{collection: "events", messages: eventsOf(cat)},
		{collection: "queries", messages: queriesOf(cat)},
	} {
		for _, msg := range group.messages {
			id := catalog.Key(msg)
			index[id] = channelMessageFM{
				Collection: group.collection,
				Name:       string(msg.Name),
				ID:         string(id) + "-" + string(msg.Version),
				Version:    string(msg.Version),
			}
		}
	}

	return index
}

