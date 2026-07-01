package catalog

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3/schema"
)

type ServiceID string

func (id ServiceID) String() string { return string(id) }

type DomainID string

func (id DomainID) String() string { return string(id) }

type MessageID string

func (id MessageID) String() string { return string(id) }

type ChannelID string

func (id ChannelID) String() string { return string(id) }

type DataStoreID string

func (id DataStoreID) String() string { return string(id) }

func (id DataStoreID) IsZero() bool { return id == "" }

type FlowID string

func (id FlowID) String() string { return string(id) }

func (id FlowID) IsZero() bool { return id == "" }

type FlowStepID string

func (id FlowStepID) String() string { return string(id) }

func (id FlowStepID) IsZero() bool { return id == "" }

type FlowEdgeID string

func (id FlowEdgeID) String() string { return string(id) }

func (id FlowEdgeID) IsZero() bool { return id == "" }

type TeamID string

func (id TeamID) String() string { return string(id) }

func (id TeamID) IsZero() bool { return id == "" }

type UserID string

func (id UserID) String() string { return string(id) }

func (id UserID) IsZero() bool { return id == "" }

type EntityID string

func (id EntityID) String() string { return string(id) }

func (id EntityID) IsZero() bool { return id == "" }

type DataProductID string

func (id DataProductID) String() string { return string(id) }

func (id DataProductID) IsZero() bool { return id == "" }

type AgentID string

func (id AgentID) String() string { return string(id) }

func (id AgentID) IsZero() bool { return id == "" }

type Direction string

func (d Direction) String() string { return string(d) }

func (d Direction) IsZero() bool { return d == "" }

type DocumentInfo struct {
	Title       string `json:"title"                 yaml:"title"`
	Version     string `json:"version"               yaml:"version"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

const (
	Sends    Direction = "sends"
	Receives Direction = "receives"
)

type MessageKind string

func (k MessageKind) String() string { return string(k) }

func (k MessageKind) IsZero() bool { return k == "" }

const (
	CommandMessage MessageKind = "command"
	EventMessage   MessageKind = "event"
	QueryMessage   MessageKind = "query"
)

type Schema = schema.Schema

type SchemaType = schema.Type

const (
	TypeString  = schema.TypeString
	TypeObject  = schema.TypeObject
	TypeInteger = schema.TypeInteger
	TypeNumber  = schema.TypeNumber
	TypeBoolean = schema.TypeBoolean
	TypeArray   = schema.TypeArray
	TypeNull    = schema.TypeNull
)

type Property = schema.Property

type Message struct {
	Kind        MessageKind       `json:"kind"`
	ID          MessageID         `json:"id"`
	Name        Name              `json:"name"`
	Version     Version           `json:"version"`
	Summary     Summary           `json:"summary,omitempty"`
	Schema      *Schema           `json:"schema,omitempty"`
	Schemas     []SchemaPointer   `json:"schemas,omitempty"`
	Direction   Direction         `json:"direction"`
	Examples    []json.RawMessage `json:"examples,omitempty"`
	Owners      []string          `json:"owners,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Deprecated  bool              `json:"deprecated,omitempty"`
	Deprecation *DeprecationInfo  `json:"deprecation,omitempty"`
	Channels    []ChannelID       `json:"channels,omitempty"`
	Changelog   []Change          `json:"changelog,omitempty"`
	Producers   []ServiceID       `json:"producers,omitempty"`
	Consumers   []ServiceID       `json:"consumers,omitempty"`
	Operation   *Operation        `json:"operation,omitempty"`
	Badges      []Badge           `json:"badges,omitempty"`
	Repository  *Repository       `json:"repository,omitempty"`
}

type Service struct {
	BaseConfig

	ID             ServiceID       `json:"id"`
	Name           Name            `json:"name"`
	Version        Version         `json:"version"`
	Summary        Summary         `json:"summary,omitempty"`
	Owners         []string        `json:"owners,omitempty"`
	Commands       []Message       `json:"commands,omitempty"`
	Events         []Message       `json:"events,omitempty"`
	Queries        []Message       `json:"queries,omitempty"`
	WritesTo       []DataStoreID   `json:"writesTo,omitempty"`
	ReadsFrom      []DataStoreID   `json:"readsFrom,omitempty"`
	Entities       []string        `json:"entities,omitempty"`
	Flows          []FlowID        `json:"flows,omitempty"`
	Repository     *Repository     `json:"repository,omitempty"`
	Badges         []Badge         `json:"badges,omitempty"`
	Specifications []Specification `json:"specifications,omitempty"`
	Attachments    []Attachment    `json:"attachments,omitempty"`
	ExternalSystem bool            `json:"externalSystem,omitempty"`
}

type Domain struct {
	BaseConfig

	ID                 DomainID                 `json:"id"`
	Name               Name                     `json:"name"`
	Version            Version                  `json:"version"`
	Summary            Summary                  `json:"summary,omitempty"`
	Owners             []string                 `json:"owners,omitempty"`
	Services           []ServiceID              `json:"services,omitempty"`
	Sends              []Ref                    `json:"sends,omitempty"`
	Receives           []Ref                    `json:"receives,omitempty"`
	Entities           []string                 `json:"entities,omitempty"`
	Flows              []FlowID                 `json:"flows,omitempty"`
	Badges             []Badge                  `json:"badges,omitempty"`
	Attachments        []Attachment             `json:"attachments,omitempty"`
	UbiquitousLanguage []UbiquitousLanguageTerm `json:"ubiquitousLanguage,omitempty"`
	SubDomains         []DomainID               `json:"subDomains,omitempty"`
	DataProducts       []DataProductID          `json:"dataProducts,omitempty"`
}

type Channel struct {
	ID                ChannelID               `json:"id"`
	Name              Name                    `json:"name"`
	Version           Version                 `json:"version"`
	Summary           Summary                 `json:"summary,omitempty"`
	Address           Address                 `json:"address,omitempty"`
	Protocols         []Protocol              `json:"protocols,omitempty"`
	Messages          []MessageID             `json:"messages,omitempty"`
	DeliveryGuarantee DeliveryGuarantee       `json:"deliveryGuarantee,omitempty"`
	Parameters        map[string]ChannelParam `json:"parameters,omitempty"`
	Routes            []ChannelRoute          `json:"routes,omitempty"`
	Owners            []string                `json:"owners,omitempty"`
	Badges            []Badge                 `json:"badges,omitempty"`
}

type Catalog struct {
	Title        Title         `json:"title"`
	Version      Version       `json:"version"`
	Services     []Service     `json:"services"`
	Domains      []Domain      `json:"domains,omitempty"`
	Channels     []Channel     `json:"channels,omitempty"`
	DataStores   []DataStore   `json:"dataStores,omitempty"`
	Flows        []Flow        `json:"flows,omitempty"`
	Teams        []Team        `json:"teams,omitempty"`
	Users        []User        `json:"users,omitempty"`
	Entities     []Entity      `json:"entities,omitempty"`
	DataProducts []DataProduct `json:"dataProducts,omitempty"`
	Agents       []Agent       `json:"agents,omitempty"`
	CustomDocs   []CustomDoc   `json:"customDocs,omitempty"`
}

// Key returns the unique key for the message: msg.ID if set, otherwise msg.Name.
// Callers that require an explicit ID should check msg.ID directly.
func Key(msg Message) MessageID {
	if msg.ID != "" {
		return msg.ID
	}

	return MessageID(msg.Name)
}
