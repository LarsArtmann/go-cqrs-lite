package asyncapi

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

// Export generates an AsyncAPI 3.0 Document from the given catalog.
func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := e.newDocument()

	if e.host != "" {
		doc.Servers = map[string]Server{
			e.serverName: { //nolint:exhaustruct
				Host: e.host, Protocol: e.protocol,
				Description: "Message broker",
			},
		}
	}

	exportMessages(e, doc, cat)

	return doc
}

func (e *Exporter) newDocument() *Document {
	return &Document{ //nolint:exhaustruct
		AsyncAPI: asyncAPIVersion,
		ID: fmt.Sprintf(
			"urn:%s:api",
			strings.ToLower(strings.ReplaceAll(e.serviceName, " ", "")),
		),
		DefaultContentType: contentType,
		Info: Info{
			Title: e.serviceName, Version: e.version, Description: e.description,
		},
		Channels:   make(map[string]Channel),
		Operations: make(map[string]Operation),
		Components: Components{
			Schemas: make(map[string]any), Messages: make(map[string]Message),
		},
	}
}

func exportMessages(e *Exporter, doc *Document, cat *catalog.Catalog) {
	for _, svc := range cat.Services {
		for _, cmd := range svc.Commands {
			e.addMessage(doc, svc.ID, cmd, kindCommand, withAction(actionReceive))
		}

		for _, evt := range svc.Events {
			action := actionSend
			if !evt.IsSend() {
				action = actionReceive
			}

			e.addMessage(doc, svc.ID, evt, kindEvent, withAction(action))
		}

		for _, q := range svc.Queries {
			e.addMessage(doc, svc.ID, q, kindQuery)
		}
	}
}

func (e *Exporter) addMessage(
	doc *Document,
	svcID catalog.ServiceID,
	msg catalog.Message,
	kind messageKind,
	opts ...messageOption,
) {
	cfg := &messageConfig{action: actionReceive}

	for _, opt := range opts {
		opt(cfg)
	}

	msgID := catalog.GetID(msg)
	channelKey := string(kind) + "." + string(msgID)
	componentKey := string(msg.Kind) + "." + string(msgID)
	ref := "#/components/messages/" + componentKey

	addChannel(doc, string(svcID), msg, kind, channelKey, ref)
	addOperation(doc, string(svcID), msg, kind, cfg, channelKey, ref, string(msgID))

	e.addMessageSchema(doc, msg)
}

func addChannel(
	doc *Document,
	svcID string,
	msg catalog.Message,
	kind messageKind,
	channelKey, ref string,
) {
	doc.Channels[channelKey] = Channel{
		Address:     fmt.Sprintf("%s.%s.%s", svcID, kind, toDotAddress(string(catalog.GetID(msg)))),
		Title:       msg.Name + " " + strings.TrimSuffix(string(kind), "s") + " Channel",
		Description: msg.Summary,
		Messages:    map[string]Ref{string(kind): {Ref: ref}},
	}
}

func addOperation(
	doc *Document,
	svcID string,
	msg catalog.Message,
	kind messageKind,
	cfg *messageConfig,
	channelKey, ref, msgID string,
) {
	opTitle, opName := operationTitleAndName(msg.Name, kind, cfg, msgID)

	doc.Operations[opName] = Operation{
		Title:    opTitle,
		Summary:  msg.Summary,
		Action:   cfg.action,
		Channel:  Ref{Ref: "#/channels/" + channelKey},
		Messages: []Ref{{Ref: ref}},
		Tags:     buildTags(kind, svcID, msg),
		Reply:    nil,
	}
}

func operationTitleAndName(
	msgName string,
	kind messageKind,
	cfg *messageConfig,
	msgID string,
) (string, string) {
	switch kind {
	case kindEvent:
		opName := "publish" + msgID
		if cfg.action == actionReceive {
			opName = actionReceive + msgID
		}

		return "Publish " + msgName, opName
	case kindCommand:
		return "Receive " + msgName, actionReceive + msgID
	case kindQuery:
		return "Handle " + msgName, "handle" + msgID
	default:
		return "Handle " + msgName, "handle" + msgID
	}
}

func kindToTagName(kind catalog.MessageKind) string {
	switch kind {
	case catalog.CommandMessage:
		return "commands"
	case catalog.EventMessage:
		return "events"
	case catalog.QueryMessage:
		return "queries"
	default:
		return "commands"
	}
}

func buildTags(kind messageKind, svcID string, msg catalog.Message) []Tag {
	tags := []Tag{{Name: string(kind)}, {Name: svcID}}

	if msg.Deprecated {
		tags = append(tags, Tag{Name: "deprecated"})
	}

	for _, owner := range msg.Owners {
		tags = append(tags, Tag{Name: "owner:" + owner})
	}

	for k, v := range msg.Labels {
		tags = append(tags, Tag{Name: k + ":" + v})
	}

	return tags
}

func (*Exporter) addMessageSchema(doc *Document, msg catalog.Message) {
	id := catalog.GetID(msg)
	componentKey := string(msg.Kind) + "." + string(id)

	doc.Components.Messages[componentKey] = Message{
		Name:        string(id),
		Title:       msg.Name,
		Summary:     msg.Summary,
		ContentType: contentType,
		Payload:     Ref{Ref: "#/components/schemas/" + componentKey},
		Tags:        []Tag{{Name: kindToTagName(msg.Kind)}},
		Examples:    toExamples(msg.Examples),
		Deprecated:  msg.Deprecated,
	}

	if msg.Schema != nil {
		doc.Components.Schemas[componentKey] = SchemaToAny(msg.Schema)
	} else {
		doc.Components.Schemas[componentKey] = SchemaToAny(nil)
	}
}
