package catalog

import "encoding/json"

// ServiceID identifies a service in the catalog (e.g., "user-service").
type ServiceID string

// String returns the underlying string value.
func (id ServiceID) String() string { return string(id) }

// DomainID identifies a business domain in the catalog (e.g., "users").
type DomainID string

// String returns the underlying string value.
func (id DomainID) String() string { return string(id) }

// MessageID identifies a message in the catalog (e.g., "CreateUser").
type MessageID string

// String returns the underlying string value.
func (id MessageID) String() string { return string(id) }

// ChannelID identifies a messaging channel in the catalog (e.g., "user.commands").
type ChannelID string

// String returns the underlying string value.
func (id ChannelID) String() string { return string(id) }

// Direction represents the flow direction of a message relative to a service.
type Direction string

const (
	// Sends indicates the service publishes this message.
	Sends Direction = "sends"
	// Receives indicates the service consumes this message.
	Receives Direction = "receives"
)

// MessageKind categorizes a message as a command, event, or query.
type MessageKind string

const (
	// CommandMessage identifies a command message.
	CommandMessage MessageKind = "command"
	// EventMessage identifies an event message.
	EventMessage MessageKind = "event"
	// QueryMessage identifies a query message.
	QueryMessage MessageKind = "query"
)

// Message describes a single command, event, or query in the catalog.
type Message struct {
	Kind      MessageKind       `json:"kind"`
	ID        MessageID         `json:"id"`
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
	ID       ServiceID `json:"id"`
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
	ID       DomainID    `json:"id"`
	Name     string      `json:"name"`
	Version  string      `json:"version"`
	Summary  string      `json:"summary,omitempty"`
	Services []ServiceID `json:"services,omitempty"`
}

// Channel represents a messaging channel used for message transport.
type Channel struct {
	ID        ChannelID   `json:"id"`
	Name      string      `json:"name"`
	Version   string      `json:"version"`
	Summary   string      `json:"summary,omitempty"`
	Address   string      `json:"address,omitempty"`
	Protocols []string    `json:"protocols,omitempty"`
	Messages  []MessageID `json:"messages,omitempty"`
}

// Catalog is an immutable snapshot of all registered services, domains, and channels.
type Catalog struct {
	Title    string    `json:"title"`
	Version  string    `json:"version"`
	Services []Service `json:"services"`
	Domains  []Domain  `json:"domains,omitempty"`
	Channels []Channel `json:"channels,omitempty"`
}

// GetID returns the ID of a message, falling back to its Name if ID is empty.
func GetID(msg Message) MessageID {
	if msg.ID != "" {
		return msg.ID
	}

	return MessageID(msg.Name)
}

// MessageIDString returns the string ID of a message, falling back to its Name if ID is empty.
//
// Deprecated: Use GetID instead for typed return.
func MessageIDString(msg Message) string {
	if msg.ID != "" {
		return string(msg.ID)
	}

	return msg.Name
}

// IsSend reports whether the message direction is Sends.
func (m Message) IsSend() bool { return m.Direction == Sends }
