package catalog

import "time"

// UbiquitousLanguageTerm represents a single term in a domain's ubiquitous language (DDD glossary).
type UbiquitousLanguageTerm struct {
	Name        Name   `json:"name"`
	Description string `json:"description,omitempty"`
}

// EntityProperty describes a single property of a domain entity.
type EntityProperty struct {
	Name                 Name   `json:"name"`
	Type                 string `json:"type"`
	Required             bool   `json:"required,omitempty"`
	Description          string `json:"description,omitempty"`
	References           Name   `json:"references,omitempty"`
	ReferencesIdentifier string `json:"referencesIdentifier,omitempty"`
	RelationType         string `json:"relationType,omitempty"`
}

// Entity represents a first-class domain entity with optional schema, owners, and badges.
type Entity struct {
	ID            EntityID         `json:"id"`
	Name          Name             `json:"name"`
	Version       Version          `json:"version"`
	Summary       Summary          `json:"summary,omitempty"`
	Schema        *Schema          `json:"schema,omitempty"`
	AggregateRoot bool             `json:"aggregateRoot,omitempty"`
	Identifier    string           `json:"identifier,omitempty"`
	Properties    []EntityProperty `json:"properties,omitempty"`
	Owners        []string         `json:"owners,omitempty"`
	Badges        []Badge          `json:"badges,omitempty"`
}

// DataProduct represents a data product in a data mesh — a curated, owned dataset.
type DataProduct struct {
	ID      DataProductID       `json:"id"`
	Name    Name                `json:"name"`
	Version Version             `json:"version"`
	Summary Summary             `json:"summary,omitempty"`
	Inputs  []Ref               `json:"inputs,omitempty"`
	Outputs []DataProductOutput `json:"outputs,omitempty"`
	Hidden  bool                `json:"hidden,omitempty"`
	Owners  []string            `json:"owners,omitempty"`
	Badges  []Badge             `json:"badges,omitempty"`
}

// Agent represents an AI agent that sends/receives messages and uses tools.
type Agent struct {
	ID        AgentID       `json:"id"`
	Name      Name          `json:"name"`
	Version   Version       `json:"version"`
	Summary   Summary       `json:"summary,omitempty"`
	Sends     []Ref         `json:"sends,omitempty"`
	Receives  []Ref         `json:"receives,omitempty"`
	ReadsFrom []DataStoreID `json:"readsFrom,omitempty"`
	WritesTo  []DataStoreID `json:"writesTo,omitempty"`
	Model     *AgentModel   `json:"model,omitempty"`
	Tools     []AgentTool   `json:"tools,omitempty"`
	Flows     []FlowID      `json:"flows,omitempty"`
	Owners    []string      `json:"owners,omitempty"`
	Badges    []Badge       `json:"badges,omitempty"`
}

// AgentModel captures the LLM provider, model, and version an agent runs on.
type AgentModel struct {
	Provider Name    `json:"provider"`
	Name     Name    `json:"name"`
	Version  Version `json:"version,omitempty"`
}

// AgentTool documents an external tool (MCP server, API, etc.) an agent can call.
type AgentTool struct {
	Name        Name        `json:"name"`
	Type        string      `json:"type,omitempty"`
	URL         URL         `json:"url,omitempty"`
	Description Description `json:"description,omitempty"`
	Icon        Icon        `json:"icon,omitempty"`
}

// DataContract represents a data contract attached to a data product output.
type DataContract struct {
	Path string `json:"path"`
	Name Name   `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// DataProductOutput extends a Ref with an optional data contract.
type DataProductOutput struct {
	Ref

	Contract *DataContract `json:"contract,omitempty"`
}

// DeprecationInfo carries structured deprecation metadata (date + message).
type DeprecationInfo struct {
	Date    *time.Time `json:"date,omitempty"`
	Message string     `json:"message,omitempty"`
}

// DataStore represents a data store (database, cache, object store, etc.).
type DataStore struct {
	ID             DataStoreID `json:"id"`
	Name           Name        `json:"name"`
	Version        Version     `json:"version"`
	Summary        Summary     `json:"summary,omitempty"`
	ContainerType  string      `json:"containerType"`
	Technology     string      `json:"technology,omitempty"`
	Classification string      `json:"classification,omitempty"`
	Retention      string      `json:"retention,omitempty"`
	Residency      string      `json:"residency,omitempty"`
	Authoritative  bool        `json:"authoritative,omitempty"`
	AccessMode     string      `json:"accessMode,omitempty"`
	Owners         []string    `json:"owners,omitempty"`
	Badges         []Badge     `json:"badges,omitempty"`
}

// Flow represents a message flow with ordered steps between services and messages.
type Flow struct {
	ID      FlowID     `json:"id"`
	Name    Name       `json:"name"`
	Version Version    `json:"version"`
	Summary Summary    `json:"summary,omitempty"`
	Steps   []FlowStep `json:"steps"`
	Badges  []Badge    `json:"badges,omitempty"`
}

// FlowStep represents a single step in a flow diagram.
// Exactly one of Service, Message, Channel, Actor, ExternalSystem, Custom,
// Agent, DataStore, DataProduct, or SubFlow should be set.
type FlowStep struct {
	ID          FlowStepID      `json:"id"`
	Title       Title           `json:"title"`
	Summary     Summary         `json:"summary,omitempty"`
	Service     *FlowStepRef    `json:"service,omitempty"`
	Message     *FlowStepRef    `json:"message,omitempty"`
	Channel     *FlowStepRef    `json:"channel,omitempty"`
	Actor       *FlowActor      `json:"actor,omitempty"`
	External    *FlowActor      `json:"external,omitempty"`
	Custom      *FlowCustomNode `json:"custom,omitempty"`
	Agent       *FlowStepRef    `json:"agent,omitempty"`
	DataStore   *FlowStepRef    `json:"dataStore,omitempty"`
	DataProduct *FlowStepRef    `json:"dataProduct,omitempty"`
	SubFlow     *FlowStepRef    `json:"subFlow,omitempty"`
	NextStep    *FlowEdge       `json:"nextStep,omitempty"`
	NextSteps   []FlowEdge      `json:"nextSteps,omitempty"`
}

// FlowStepRef references a catalog resource by ID and optional version.
type FlowStepRef = Ref

// FlowActor describes a person or external system in a flow step.
type FlowActor struct {
	Name    Name    `json:"name"`
	Summary Summary `json:"summary,omitempty"`
	URL     URL     `json:"url,omitempty"`
}

// FlowCustomNode describes a custom node type in a flow step.
type FlowCustomNode struct {
	Title   Title   `json:"title"`
	Icon    Icon    `json:"icon,omitempty"`
	Type    string  `json:"type,omitempty"`
	Summary Summary `json:"summary,omitempty"`
	URL     URL     `json:"url,omitempty"`
	Color   Color   `json:"color,omitempty"`
}

// FlowEdge connects one step to the next with an optional label.
type FlowEdge struct {
	ID    FlowEdgeID `json:"id"`
	Label string     `json:"label,omitempty"`
}

// Team represents a team that owns catalog resources.
type Team struct {
	ID                    TeamID   `json:"id"`
	Name                  Name     `json:"name"`
	Summary               Summary  `json:"summary,omitempty"`
	Members               []string `json:"members,omitempty"`
	Email                 Email    `json:"email,omitempty"`
	SlackDirectMessageURL URL      `json:"slackDirectMessageUrl,omitempty"`
}

// User represents an individual who owns catalog resources.
type User struct {
	ID                    UserID `json:"id"`
	Name                  Name   `json:"name"`
	AvatarURL             URL    `json:"avatarUrl,omitempty"`
	Role                  Role   `json:"role,omitempty"`
	Email                 Email  `json:"email,omitempty"`
	SlackDirectMessageURL URL    `json:"slackDirectMessageUrl,omitempty"`
}

// CustomDoc represents a global custom documentation page (ADRs, architecture docs, etc.).
type CustomDoc struct {
	ID      CustomDocID `json:"id"`
	Title   Title       `json:"title"`
	Summary Summary     `json:"summary,omitempty"`
	Slug    string      `json:"slug,omitempty"`
	Content string      `json:"content,omitempty"`
	Owners  []string    `json:"owners,omitempty"`
	Badges  []Badge     `json:"badges,omitempty"`
}

// CustomDocID is a branded identifier for a custom documentation page.
type CustomDocID string

func (id CustomDocID) String() string { return string(id) }

func (id CustomDocID) IsZero() bool { return id == "" }
