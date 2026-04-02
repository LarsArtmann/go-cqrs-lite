package asyncapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/yaml"
)

type Exporter struct {
	ServiceName string
	Version     string
	Description string
	Protocol    string
	Host        string
}

func NewExporter(serviceName, version string) *Exporter {
	return &Exporter{
		ServiceName: serviceName,
		Version:     version,
		Protocol:    "kafka",
		Host:        "localhost:9092",
	}
}

func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := &Document{
		AsyncAPI:           "3.0.0",
		ID:                 fmt.Sprintf("urn:%s:api", strings.ToLower(strings.ReplaceAll(e.ServiceName, " ", ""))),
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
			"production": {
				Host:        e.Host,
				Protocol:    e.Protocol,
				Description: "Message broker",
			},
		}
	}

	for _, svc := range cat.Services {
		for _, cmd := range svc.Commands {
			e.addCommand(doc, svc.ID, cmd)
		}
		for _, evt := range svc.Events {
			e.addEvent(doc, svc.ID, evt)
		}
		for _, q := range svc.Queries {
			e.addQuery(doc, svc.ID, q)
		}
	}

	return doc
}

func (e *Exporter) addCommand(doc *Document, svcID string, msg catalog.Message) {
	id := msg.ID
	if id == "" {
		id = string(msg.Name)
	}

	channelKey := "commands." + id
	ref := "#/components/messages/" + id

	doc.Channels[channelKey] = Channel{
		Address:     fmt.Sprintf("%s.commands.%s", svcID, toDotAddress(id)),
		Title:       msg.Name + " Command Channel",
		Description: msg.Summary,
		Messages:    map[string]Ref{"command": {Ref: ref}},
	}

	doc.Operations["receive"+id] = Operation{
		Title:    "Receive " + msg.Name,
		Summary:  msg.Summary,
		Action:   "receive",
		Channel:  Ref{Ref: "#/channels/" + channelKey},
		Messages: []Ref{{Ref: ref}},
		Tags:     []Tag{{Name: "commands"}, {Name: svcID}},
	}

	e.addMessageSchema(doc, msg)
}

func (e *Exporter) addEvent(doc *Document, svcID string, msg catalog.Message) {
	id := msg.ID
	if id == "" {
		id = string(msg.Name)
	}

	action := "send"
	if msg.Direction == catalog.Receives {
		action = "receive"
	}

	channelKey := "events." + id
	ref := "#/components/messages/" + id

	doc.Channels[channelKey] = Channel{
		Address:     fmt.Sprintf("%s.events.%s", svcID, toDotAddress(id)),
		Title:       msg.Name + " Event Channel",
		Description: msg.Summary,
		Messages:    map[string]Ref{"event": {Ref: ref}},
	}

	opName := "publish" + id
	if action == "receive" {
		opName = "receive" + id
	}

	doc.Operations[opName] = Operation{
		Title:    opName,
		Summary:  msg.Summary,
		Action:   action,
		Channel:  Ref{Ref: "#/channels/" + channelKey},
		Messages: []Ref{{Ref: ref}},
		Tags:     []Tag{{Name: "events"}, {Name: svcID}},
	}

	e.addMessageSchema(doc, msg)
}

func (e *Exporter) addQuery(doc *Document, svcID string, msg catalog.Message) {
	id := msg.ID
	if id == "" {
		id = string(msg.Name)
	}

	channelKey := "queries." + id
	ref := "#/components/messages/" + id

	doc.Channels[channelKey] = Channel{
		Address:     fmt.Sprintf("%s.queries.%s", svcID, toDotAddress(id)),
		Title:       msg.Name + " Query Channel",
		Description: msg.Summary,
		Messages:    map[string]Ref{"query": {Ref: ref}},
	}

	doc.Operations["handle"+id] = Operation{
		Title:    "Handle " + msg.Name,
		Summary:  msg.Summary,
		Action:   "receive",
		Channel:  Ref{Ref: "#/channels/" + channelKey},
		Messages: []Ref{{Ref: ref}},
		Tags:     []Tag{{Name: "queries"}, {Name: svcID}},
	}

	e.addMessageSchema(doc, msg)
}

func (*Exporter) addMessageSchema(doc *Document, msg catalog.Message) {
	id := msg.ID
	if id == "" {
		id = string(msg.Name)
	}

	tagName := "commands"
	switch msg.Kind {
	case catalog.EventMessage:
		tagName = "events"
	case catalog.QueryMessage:
		tagName = "queries"
	}

	doc.Components.Messages[id] = Message{
		Name:        id,
		Title:       msg.Name,
		Summary:     msg.Summary,
		ContentType: "application/json",
		Payload:     Ref{Ref: "#/components/schemas/" + id},
		Tags:        []Tag{{Name: tagName}},
	}

	if msg.Schema != nil {
		doc.Components.Schemas[id] = schemaToAny(msg.Schema)
	} else {
		doc.Components.Schemas[id] = map[string]string{"type": "object"}
	}
}

func schemaToAny(s *catalog.Schema) any {
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

func (d *Document) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(d)
}

func (d *Document) MarshalJSON() ([]byte, error) {
	type alias Document
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
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
