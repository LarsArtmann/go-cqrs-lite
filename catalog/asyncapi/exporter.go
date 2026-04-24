package asyncapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-faster/yaml"
	"github.com/larsartmann/go-cqrs-lite/catalog"
)

type messageKind string

const (
	kindCommand messageKind = "commands"
	kindEvent   messageKind = "events"
	kindQuery   messageKind = "queries"
)

type Exporter struct {
	ServiceName string
	Version     string
	Description string
	Protocol    string
	Host        string
	ServerName  string
}

type Option func(*Exporter)

func WithServer(name, host, protocol string) Option {
	return func(e *Exporter) {
		e.ServerName = name
		e.Host = host
		e.Protocol = protocol
	}
}

func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.Description = desc
	}
}

func NewExporter(serviceName, version string, opts ...Option) *Exporter {
	e := &Exporter{
		ServiceName: serviceName,
		Version:     version,
		Protocol:    "kafka",
		Host:        "localhost:9092",
		ServerName:  "production",
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := &Document{
		AsyncAPI: "3.0.0",
		ID: fmt.Sprintf(
			"urn:%s:api",
			strings.ToLower(strings.ReplaceAll(e.ServiceName, " ", "")),
		),
		DefaultContentType: "application/json",
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
				Host:        e.Host,
				Protocol:    e.Protocol,
				Description: "Message broker",
			},
		}
	}

	for _, svc := range cat.Services {
		for _, cmd := range svc.Commands {
			e.addMessage(doc, svc.ID, cmd, kindCommand)
		}

		for _, evt := range svc.Events {
			action := "send"
			if evt.Direction == catalog.Receives {
				action = "receive"
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
	cfg := &messageConfig{action: "receive"}
	for _, opt := range opts {
		opt(cfg)
	}

	id := MessageID(msg)
	channelKey := string(kind) + "." + id
	ref := "#/components/messages/" + id

	doc.Channels[channelKey] = Channel{
		Address:     fmt.Sprintf("%s.%s.%s", svcID, kind, toDotAddress(id)),
		Title:       msg.Name + " " + strings.TrimSuffix(string(kind), "s") + " Channel",
		Description: msg.Summary,
		Messages:    map[string]Ref{string(kind): {Ref: ref}},
	}

	opTitle := "Handle " + msg.Name
	opName := "handle" + id

	switch kind {
	case kindEvent:
		opTitle = "Publish " + msg.Name

		opName = "publish" + id
		if cfg.action == "receive" {
			opName = "receive" + id
		}
	case kindCommand:
		opTitle = "Receive " + msg.Name
		opName = "receive" + id
	case kindQuery:
		opTitle = "Handle " + msg.Name
		opName = "handle" + id
	}

	doc.Operations[opName] = Operation{
		Title:    opTitle,
		Summary:  msg.Summary,
		Action:   cfg.action,
		Channel:  Ref{Ref: "#/channels/" + channelKey},
		Messages: []Ref{{Ref: ref}},
		Tags:     []Tag{{Name: string(kind)}, {Name: svcID}},
	}

	e.addMessageSchema(doc, msg)
}

func (*Exporter) addMessageSchema(doc *Document, msg catalog.Message) {
	id := MessageID(msg)

	tagName := "commands"

	switch msg.Kind {
	case catalog.EventMessage:
		tagName = "events"
	case catalog.QueryMessage:
		tagName = "queries"
	case catalog.CommandMessage:
		tagName = "commands"
	}

	doc.Components.Messages[id] = Message{
		Name:        id,
		Title:       msg.Name,
		Summary:     msg.Summary,
		ContentType: "application/json",
		Payload:     Ref{Ref: "#/components/schemas/" + id},
		Tags:        []Tag{{Name: tagName}},
		Examples:    toExamples(msg.Examples),
	}

	if msg.Schema != nil {
		doc.Components.Schemas[id] = SchemaToAny(msg.Schema)
	} else {
		doc.Components.Schemas[id] = map[string]string{"type": "object"}
	}
}

func SchemaToAny(s *catalog.Schema) any {
	if s == nil {
		return map[string]string{"type": "object"}
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return map[string]string{"type": "object"}
	}

	var result any

	_ = json.Unmarshal(raw, &result)

	return result
}

func MessageID(msg catalog.Message) string {
	if msg.ID != "" {
		return msg.ID
	}

	return msg.Name
}

func (d *Document) MarshalYAML() ([]byte, error) {
	//nolint:wrapcheck // MarshalYAML returns bytes; caller handles error
	return yaml.Marshal(d)
}

func (d *Document) MarshalJSON() ([]byte, error) {
	type alias Document
	//nolint:wrapcheck // MarshalJSON returns bytes; caller handles error
	return json.MarshalIndent((*alias)(d), "", "  ")
}

func toDotAddress(s string) string {
	var result []byte

	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '.')
			}

			result = append(result, byte(c+'a'-'A'))
		} else if c >= 0 && c <= 127 {
			result = append(result, byte(c))
		}
	}

	return string(result)
}

func toExamples(raw []json.RawMessage) []Example {
	if len(raw) == 0 {
		return nil
	}

	examples := make([]Example, len(raw))

	for i, r := range raw {
		examples[i] = Example{Payload: r}
	}

	return examples
}
