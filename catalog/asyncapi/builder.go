package asyncapi

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

// Export generates an AsyncAPI 3.0 Document from the given catalog.
func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := e.newDocument()

	if e.host != "" {
		doc.Servers = map[string]Server{
			e.serverName: { //nolint:exhaustruct // optional fields omitted by design
				Host: e.host, Protocol: e.protocol,
				Description: "Message broker",
			},
		}
	}

	exportMessages(e, doc, cat)

	return doc
}

func (e *Exporter) newDocument() *Document {
	return &Document{ //nolint:exhaustruct // optional fields omitted by design
		AsyncAPI: asyncAPIVersion,
		ID: URI(fmt.Sprintf(
			"urn:%s:api",
			strings.ToLower(strings.ReplaceAll(e.serviceName, " ", "")),
		)),
		DefaultContentType: contentType,
		Info: catalog.DocumentInfo{
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
	serviceID catalog.ServiceID,
	msg catalog.Message,
	kind messageKind,
	opts ...messageOption,
) {
	cfg := &messageConfig{action: actionReceive}

	for _, opt := range opts {
		opt(cfg)
	}

	messageID := catalog.Key(msg)
	channelKey := string(kind) + "." + string(messageID)
	componentKey := string(msg.Kind) + "." + string(messageID)
	ref := "#/components/messages/" + componentKey

	addChannel(doc, serviceID, msg, kind, channelKey, ref)
	addOperation(doc, serviceID, msg, kind, cfg, channelKey, ref, messageID)

	e.addMessageSchema(doc, msg)
}

func addChannel(
	doc *Document,
	serviceID catalog.ServiceID,
	msg catalog.Message,
	kind messageKind,
	channelKey, ref string,
) {
	doc.Channels[channelKey] = Channel{
		Address: fmt.Sprintf(
			"%s.%s.%s",
			serviceID,
			kind,
			dotSeparated(string(catalog.Key(msg))),
		),
		Title:       string(msg.Name) + " " + strings.TrimSuffix(string(kind), "s") + " Channel",
		Description: string(msg.Summary),
		Messages:    map[string]Ref{string(kind): {Ref: ref}},
	}
}

func addOperation(
	doc *Document,
	serviceID catalog.ServiceID,
	msg catalog.Message,
	kind messageKind,
	cfg *messageConfig,
	channelKey, ref string,
	messageID catalog.MessageID,
) {
	opTitle, opName := operationTitleAndName(string(msg.Name), kind, cfg, messageID)

	doc.Operations[opName] = Operation{
		Title:    opTitle,
		Summary:  string(msg.Summary),
		Action:   cfg.action,
		Channel:  Ref{Ref: "#/channels/" + channelKey},
		Messages: []Ref{{Ref: ref}},
		Tags:     buildTags(kind, serviceID, msg),
		Reply:    nil,
	}
}

func operationTitleAndName(
	msgName string,
	kind messageKind,
	cfg *messageConfig,
	messageID catalog.MessageID,
) (string, string) {
	switch kind {
	case kindEvent:
		opName := "publish" + string(messageID)
		if cfg.action == actionReceive {
			opName = actionReceive + string(messageID)
		}

		return "Publish " + msgName, opName
	case kindCommand:
		return "Receive " + msgName, actionReceive + string(messageID)
	case kindQuery:
		return "Handle " + msgName, "handle" + string(messageID)
	default:
		return "Handle " + msgName, "handle" + string(messageID)
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

func buildTags(kind messageKind, serviceID catalog.ServiceID, msg catalog.Message) []Tag {
	tags := []Tag{{Name: string(kind)}, {Name: string(serviceID)}}

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
	messageID := catalog.Key(msg)
	componentKey := string(msg.Kind) + "." + string(messageID)

	doc.Components.Messages[componentKey] = Message{
		Name:        string(messageID),
		Title:       string(msg.Name),
		Summary:     string(msg.Summary),
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
