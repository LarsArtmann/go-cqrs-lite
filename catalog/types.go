package catalog

import "encoding/json"

type Direction string

const (
	Sends    Direction = "sends"
	Receives Direction = "receives"
)

type MessageKind string

const (
	CommandMessage MessageKind = "command"
	EventMessage   MessageKind = "event"
	QueryMessage   MessageKind = "query"
)

type Message struct {
	Kind      MessageKind       `json:"kind"`
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Summary   string            `json:"summary,omitempty"`
	Schema    *Schema           `json:"schema,omitempty"`
	Direction Direction         `json:"direction"`
	Examples  []json.RawMessage `json:"examples,omitempty"`
}

type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
	Items      *Property           `json:"items,omitempty"`
}

type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Format      string              `json:"format,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Required    []string            `json:"required,omitempty"`
}

type Service struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Version  string    `json:"version"`
	Summary  string    `json:"summary,omitempty"`
	Commands []Message `json:"commands,omitempty"`
	Events   []Message `json:"events,omitempty"`
	Queries  []Message `json:"queries,omitempty"`
}

type Domain struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Summary  string   `json:"summary,omitempty"`
	Services []string `json:"services,omitempty"`
}

type Channel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Summary   string   `json:"summary,omitempty"`
	Address   string   `json:"address,omitempty"`
	Protocols []string `json:"protocols,omitempty"`
	Messages  []string `json:"messages,omitempty"`
}

type Catalog struct {
	Title    string    `json:"title"`
	Version  string    `json:"version"`
	Services []Service `json:"services"`
	Domains  []Domain  `json:"domains,omitempty"`
	Channels []Channel `json:"channels,omitempty"`
}
