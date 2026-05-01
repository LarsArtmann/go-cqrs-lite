package catalog

import "encoding/json"

// Direction represents the flow direction of a message relative to a service.
type Direction string

const (
	Sends    Direction = "sends"
	Receives Direction = "receives"
)

// MessageKind categorizes a message as a command, event, or query.
type MessageKind string

const (
	CommandMessage MessageKind = "command"
	EventMessage   MessageKind = "event"
	QueryMessage   MessageKind = "query"
)

// Message describes a single command, event, or query in the catalog.
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

// Schema represents a JSON Schema object with properties, required fields, and items.
type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
	Items      *Property           `json:"items,omitempty"`
}

// Property describes a single field within a JSON Schema.
type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Format      string              `json:"format,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Default     string              `json:"default,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Nullable    bool                `json:"nullable,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty"`
	Pattern     string              `json:"pattern,omitempty"`
}

// Service groups related commands, events, and queries under a logical service.
type Service struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Version  string    `json:"version"`
	Summary  string    `json:"summary,omitempty"`
	Owners   []string  `json:"owners,omitempty"`
	Commands []Message `json:"commands,omitempty"`
	Events   []Message `json:"events,omitempty"`
	Queries  []Message `json:"queries,omitempty"`
}

// Domain represents a business domain that groups multiple services.
type Domain struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Summary  string   `json:"summary,omitempty"`
	Services []string `json:"services,omitempty"`
}

// Channel represents a messaging channel used for message transport.
type Channel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Summary   string   `json:"summary,omitempty"`
	Address   string   `json:"address,omitempty"`
	Protocols []string `json:"protocols,omitempty"`
	Messages  []string `json:"messages,omitempty"`
}

// Catalog is an immutable snapshot of all registered services, domains, and channels.
type Catalog struct {
	Title    string    `json:"title"`
	Version  string    `json:"version"`
	Services []Service `json:"services"`
	Domains  []Domain  `json:"domains,omitempty"`
	Channels []Channel `json:"channels,omitempty"`
}

// MessageID returns the ID of a message, falling back to its Name if ID is empty.
func MessageID(msg Message) string {
	if msg.ID != "" {
		return msg.ID
	}

	return msg.Name
}

// IsSend reports whether the message direction is Sends.
func (m Message) IsSend() bool { return m.Direction == Sends }
