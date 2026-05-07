package asyncapi

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	asyncAPIVersion = "3.0.0"
	contentType     = "application/json"
	typeObject      = "object"

	actionSend    = "send"
	actionReceive = "receive"
)

type messageKind string

const (
	kindCommand messageKind = "commands"
	kindEvent   messageKind = "events"
	kindQuery   messageKind = "queries"
)

// Exporter generates an AsyncAPI 3.0 document from a catalog.
type Exporter struct {
	ServiceName string
	Version     string
	Description string
	Protocol    string
	Host        string
	ServerName  string
}

// Option configures an Exporter.
type Option func(*Exporter)

// WithServer sets the server name, host, and protocol for the AsyncAPI document.
func WithServer(name, host, protocol string) Option {
	return func(e *Exporter) {
		e.ServerName = name
		e.Host = host
		e.Protocol = protocol
	}
}

// WithDescription sets the description for the AsyncAPI document.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.Description = desc
	}
}

// NewExporter creates an AsyncAPI exporter with the given service name and version.
func NewExporter(serviceName, version string, opts ...Option) *Exporter {
	e := &Exporter{
		ServiceName: serviceName,
		Version:     version,
		Description: "",
		Protocol:    "kafka",
		Host:        "localhost:9092",
		ServerName:  "production",
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Export generates an AsyncAPI 3.0 Document from the given catalog.
func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := &Document{
		AsyncAPI: asyncAPIVersion,
		ID: fmt.Sprintf(
			"urn:%s:api",
			strings.ToLower(strings.ReplaceAll(e.ServiceName, " ", "")),
		),
		DefaultContentType: contentType,
		Servers:            nil,
		Info: Info{
			Title:       e.ServiceName,
			Version:     e.Version,
			Description: e.Description,
		},
		Channels:   make(map[string]Channel),
		Operations: make(map[string]Operation),
		Components: Components{
			Schemas:  make(map[string]any),
			Messages: make(map[string]Message),
		},
	}

	if e.Host != "" {
		doc.Servers = map[string]Server{
			e.ServerName: {
				Host:            e.Host,
				Protocol:        e.Protocol,
				ProtocolVersion: "",
				Description:     "Message broker",
				Tags:            nil,
			},
		}
	}

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

	return doc
}

type messageOption func(*messageConfig)

type messageConfig struct {
	action string
}

func withAction(action string) messageOption {
	return func(c *messageConfig) {
		c.action = action
	}
}

func (e *Exporter) addMessage(
	doc *Document,
	svcID string,
	msg catalog.Message,
	kind messageKind,
	opts ...messageOption,
) {
	cfg := &messageConfig{action: actionReceive}

	for _, opt := range opts {
		opt(cfg)
	}

	msgID := catalog.MessageID(msg)
	channelKey := string(kind) + "." + msgID
	componentKey := string(msg.Kind) + "." + msgID
	ref := "#/components/messages/" + componentKey

	addChannel(doc, svcID, msg, kind, channelKey, ref)
	addOperation(doc, svcID, msg, kind, cfg, channelKey, ref, msgID)

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
		Address:     fmt.Sprintf("%s.%s.%s", svcID, kind, toDotAddress(catalog.MessageID(msg))),
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
		Tags:     []Tag{{Name: string(kind)}, {Name: svcID}},
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

func (*Exporter) addMessageSchema(doc *Document, msg catalog.Message) {
	id := catalog.MessageID(msg)
	componentKey := string(msg.Kind) + "." + id

	doc.Components.Messages[componentKey] = Message{
		Name:        id,
		Title:       msg.Name,
		Summary:     msg.Summary,
		ContentType: contentType,
		Payload:     Ref{Ref: "#/components/schemas/" + componentKey},
		Tags:        []Tag{{Name: kindToTagName(msg.Kind)}},
		Examples:    toExamples(msg.Examples),
	}

	if msg.Schema != nil {
		doc.Components.Schemas[componentKey] = SchemaToAny(msg.Schema)
	} else {
		doc.Components.Schemas[componentKey] = map[string]string{"type": typeObject}
	}
}
