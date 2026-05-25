package catalog

import (
	"encoding/json"
	"time"
)

// ServiceID identifies a service in the catalog (e.g., "user-service").
type ServiceID string

// SchemaType represents a JSON Schema type value (e.g., "string", "object", "array").
type SchemaType string

const (
	// TypeString represents the JSON Schema "string" type.
	TypeString SchemaType = "string"
	// TypeObject represents the JSON Schema "object" type.
	TypeObject SchemaType = "object"
	// TypeInteger represents the JSON Schema "integer" type.
	TypeInteger SchemaType = "integer"
	// TypeNumber represents the JSON Schema "number" type.
	TypeNumber SchemaType = "number"
	// TypeBoolean represents the JSON Schema "boolean" type.
	TypeBoolean SchemaType = "boolean"
	// TypeArray represents the JSON Schema "array" type.
	TypeArray SchemaType = "array"
	// TypeNull represents the JSON Schema "null" type.
	TypeNull SchemaType = "null"
)

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
	Kind       MessageKind       `json:"kind"`
	ID         MessageID         `json:"id"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Summary    string            `json:"summary,omitempty"`
	Schema     *Schema           `json:"schema,omitempty"`
	Direction  Direction         `json:"direction"`
	Examples   []json.RawMessage `json:"examples,omitempty"`
	Owners     []string          `json:"owners,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Deprecated bool              `json:"deprecated,omitempty"`
	Changelog  []Change          `json:"changelog,omitempty"`
	Producers  []string          `json:"producers,omitempty"`
	Consumers  []string          `json:"consumers,omitempty"`
	Operation  *Operation        `json:"operation,omitempty"`
	Badges     []Badge           `json:"badges,omitempty"`
	Repository *Repository       `json:"repository,omitempty"`
}

// Schema represents a JSON Schema object with properties, required fields, and items.
type Schema struct {
	Type       SchemaType          `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
	Items      *Property           `json:"items,omitempty"`
	Examples   []json.RawMessage   `json:"examples,omitempty"`
}

// Property describes a single field within a JSON Schema.
type Property struct {
	Type        SchemaType          `json:"type"`
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
	Examples    []json.RawMessage   `json:"examples,omitempty"`
}

// Service groups related commands, events, and queries under a logical service.
type Service struct {
	ID             ServiceID       `json:"id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Summary        string          `json:"summary,omitempty"`
	Owners         []string        `json:"owners,omitempty"`
	Commands       []Message       `json:"commands,omitempty"`
	Events         []Message       `json:"events,omitempty"`
	Queries        []Message       `json:"queries,omitempty"`
	WritesTo       []string        `json:"writesTo,omitempty"`
	ReadsFrom      []string        `json:"readsFrom,omitempty"`
	Entities       []string        `json:"entities,omitempty"`
	Flows          []string        `json:"flows,omitempty"`
	Repository     *Repository     `json:"repository,omitempty"`
	Badges         []Badge         `json:"badges,omitempty"`
	Specifications []Specification `json:"specifications,omitempty"`
	Attachments    []Attachment    `json:"attachments,omitempty"`
}

// Domain represents a business domain that groups multiple services.
type Domain struct {
	ID          DomainID         `json:"id"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Summary     string           `json:"summary,omitempty"`
	Owners      []string         `json:"owners,omitempty"`
	Services    []ServiceID      `json:"services,omitempty"`
	Sends       []Ref `json:"sends,omitempty"`
	Receives    []Ref `json:"receives,omitempty"`
	Entities    []string         `json:"entities,omitempty"`
	Flows       []string         `json:"flows,omitempty"`
	Badges      []Badge          `json:"badges,omitempty"`
	Attachments []Attachment     `json:"attachments,omitempty"`
}

// Channel represents a messaging channel used for message transport.
type Channel struct {
	ID               ChannelID                `json:"id"`
	Name             string                   `json:"name"`
	Version          string                   `json:"version"`
	Summary          string                   `json:"summary,omitempty"`
	Address          string                   `json:"address,omitempty"`
	Protocols        []string                 `json:"protocols,omitempty"`
	Messages         []MessageID              `json:"messages,omitempty"`
	DeliveryGuarantee string                 `json:"deliveryGuarantee,omitempty"`
	Parameters       map[string]ChannelParam  `json:"parameters,omitempty"`
	Routes           []ChannelRoute           `json:"routes,omitempty"`
	Owners           []string                 `json:"owners,omitempty"`
	Badges           []Badge                  `json:"badges,omitempty"`
}

// Catalog is an immutable snapshot of all registered resources.
type Catalog struct {
	Title      string      `json:"title"`
	Version    string      `json:"version"`
	Services   []Service   `json:"services"`
	Domains    []Domain    `json:"domains,omitempty"`
	Channels   []Channel   `json:"channels,omitempty"`
	DataStores []DataStore `json:"dataStores,omitempty"`
	Flows      []Flow      `json:"flows,omitempty"`
	Teams      []Team      `json:"teams,omitempty"`
	Users      []User      `json:"users,omitempty"`
}

// GetID returns the ID of a message, falling back to its Name if ID is empty.
func GetID(msg Message) MessageID {
	if msg.ID != "" {
		return msg.ID
	}

	return MessageID(msg.Name)
}

// Change describes a single modification to a message over time.
type Change struct {
	Version string     `json:"version"`
	Date    *time.Time `json:"date,omitempty"`
	Summary string     `json:"summary"`
}

// IsSend reports whether the message direction is Sends.
func (m Message) IsSend() bool { return m.Direction == Sends }

// Badge represents a visual badge rendered on a catalog resource.
type Badge struct {
	Content         string `json:"content"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	TextColor       string `json:"textColor,omitempty"`
	Icon            string `json:"icon,omitempty"`
	URL             string `json:"url,omitempty"`
}

// Repository describes a code repository associated with a resource.
type Repository struct {
	Language string `json:"language,omitempty"`
	URL      string `json:"url,omitempty"`
}

// Operation maps a message to an HTTP endpoint.
type Operation struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	StatusCodes []string `json:"statusCodes,omitempty"`
}

// Specification describes an API specification attached to a service.
type Specification struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// Attachment links to an external resource (ADR, runbook, diagram, etc.).
type Attachment struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// Ref references a catalog resource by ID and optional version.
type Ref struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

// ChannelParam describes a parameter for dynamic channel addressing.
type ChannelParam struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
}

// ChannelRoute describes a routing rule from one channel to another.
type ChannelRoute struct {
	ID string   `json:"id"`
	To []string `json:"to,omitempty"`
}
