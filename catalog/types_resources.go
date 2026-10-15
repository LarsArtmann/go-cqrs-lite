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
	Schemas       []SchemaPointer  `json:"schemas,omitempty"`
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
	Owners  []string   `json:"owners,omitempty"`
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

// ADRStatus is the lifecycle state of an architecture decision record.
// The vocabulary tracks the EventCatalog ADR schema exactly; the exporter
// rejects anything outside it.
type ADRStatus string

const (
	ADRProposed   ADRStatus = "proposed"
	ADRAccepted   ADRStatus = "accepted"
	ADRRejected   ADRStatus = "rejected"
	ADRDeprecated ADRStatus = "deprecated"
	ADRSuperseded ADRStatus = "superseded"
)

// ADRResourceType names the catalog resource kind an ADR applies to.
// The vocabulary tracks the EventCatalog appliesTo enum exactly; the
// exporter rejects anything outside it.
type ADRResourceType string

const (
	ADRAppliesToAgent       ADRResourceType = "agent"
	ADRAppliesToService     ADRResourceType = "service"
	ADRAppliesToEvent       ADRResourceType = "event"
	ADRAppliesToCommand     ADRResourceType = "command"
	ADRAppliesToQuery       ADRResourceType = "query"
	ADRAppliesToFlow        ADRResourceType = "flow"
	ADRAppliesToChannel     ADRResourceType = "channel"
	ADRAppliesToDomain      ADRResourceType = "domain"
	ADRAppliesToSystem      ADRResourceType = "system"
	ADRAppliesToUser        ADRResourceType = "user"
	ADRAppliesToTeam        ADRResourceType = "team"
	ADRAppliesToContainer   ADRResourceType = "container"
	ADRAppliesToEntity      ADRResourceType = "entity"
	ADRAppliesToDiagram     ADRResourceType = "diagram"
	ADRAppliesToDataProduct ADRResourceType = "data-product"
)

// ADR represents an architecture decision record: a dated, status-carrying
// decision that applies to catalog resources and may supersede, amend, or
// relate to other records. Body carries the decision prose
// (Context / Decision / Consequences); it renders below the page heading.
// Date is the calendar date of the decision in YYYY-MM-DD form.
type ADR struct {
	ID             ADRID            `json:"id"`
	Name           Name             `json:"name"`
	Version        Version          `json:"version"`
	Status         ADRStatus        `json:"status"`
	Date           string           `json:"date"`
	Summary        Summary          `json:"summary,omitempty"`
	Body           string           `json:"body,omitempty"`
	Owners         []string         `json:"owners,omitempty"`
	DecisionMakers []string         `json:"decisionMakers,omitempty"`
	AppliesTo      []ADRResourceRef `json:"appliesTo,omitempty"`
	Supersedes     []ADRRef         `json:"supersedes,omitempty"`
	SupersededBy   []ADRRef         `json:"supersededBy,omitempty"`
	Amends         []ADRRef         `json:"amends,omitempty"`
	AmendedBy      []ADRRef         `json:"amendedBy,omitempty"`
	Related        []ADRRef         `json:"related,omitempty"`
	Badges         []Badge          `json:"badges,omitempty"`
}

// ADRRef references another ADR by ID with an optional version pin.
type ADRRef struct {
	ID      ADRID   `json:"id"`
	Version Version `json:"version,omitempty"`
}

// ADRResourceRef points at the catalog resource a decision applies to.
// Type selects the resource kind; Version optionally pins a version
// (unpinned pointers resolve to the resource's latest version).
type ADRResourceRef struct {
	Type    ADRResourceType `json:"type"`
	ID      string          `json:"id"`
	Version Version         `json:"version,omitempty"`
}

// SchemaPointer references a schema file by path, format, and optional environment.
type SchemaPointer struct {
	ID      string `json:"id,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	File    string `json:"file,omitempty"`
	Path    string `json:"path,omitempty"`
	Name    Name   `json:"name,omitempty"`
	Format  string `json:"format,omitempty"`
	Default bool   `json:"default,omitempty"`
}

// SidebarConfig customizes how a resource appears in EventCatalog's sidebar.
type SidebarConfig struct {
	Badge string `json:"badge,omitempty"`
	Label string `json:"label,omitempty"`
}

// StylesConfig customizes visual styling for a resource in EventCatalog.
type StylesConfig struct {
	Icon      string `json:"icon,omitempty"`
	NodeColor string `json:"nodeColor,omitempty"`
	NodeLabel string `json:"nodeLabel,omitempty"`
}

// DraftConfig marks a resource as draft with a title and message.
type DraftConfig struct {
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
}

// ResourceGroup groups related resources in EventCatalog's UI.
type ResourceGroup struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Items []string `json:"items,omitempty"`
	Limit int      `json:"limit,omitempty"`
}

// DetailsPanelConfig controls visibility of detail panel sections.
type DetailsPanelConfig struct {
	Sections []string `json:"sections,omitempty"`
}

// BaseConfig holds shared EventCatalog UI configuration fields.
// Embedded in Service and Domain for convenience.
type BaseConfig struct {
	Sidebar        *SidebarConfig      `json:"sidebar,omitempty"`
	Styles         *StylesConfig       `json:"styles,omitempty"`
	EditUrl        string              `json:"editUrl,omitempty"`
	Draft          *DraftConfig        `json:"draft,omitempty"`
	Visualiser     *bool               `json:"visualiser,omitempty"`
	ResourceGroups []ResourceGroup     `json:"resourceGroups,omitempty"`
	DetailsPanel   *DetailsPanelConfig `json:"detailsPanel,omitempty"`
}

// Source represents an external system that syncs ownership data (GitHub, Azure AD, etc.).
type Source struct {
	Provider string `json:"provider"`
	ID       string `json:"id,omitempty"`
	URL      URL    `json:"url,omitempty"`
}

// Team represents a team that owns catalog resources.
type Team struct {
	ID                    TeamID   `json:"id"`
	Name                  Name     `json:"name"`
	Summary               Summary  `json:"summary,omitempty"`
	Members               []string `json:"members,omitempty"`
	Email                 Email    `json:"email,omitempty"`
	AvatarURL             URL      `json:"avatarUrl,omitempty"`
	Role                  Role     `json:"role,omitempty"`
	SlackDirectMessageURL URL      `json:"slackDirectMessageUrl,omitempty"`
	Hidden                bool     `json:"hidden,omitempty"`
	ReadOnly              bool     `json:"readOnly,omitempty"`
	Source                *Source  `json:"source,omitempty"`
}

// User represents an individual who owns catalog resources.
type User struct {
	ID                    UserID  `json:"id"`
	Name                  Name    `json:"name"`
	AvatarURL             URL     `json:"avatarUrl,omitempty"`
	Role                  Role    `json:"role,omitempty"`
	Email                 Email   `json:"email,omitempty"`
	SlackDirectMessageURL URL     `json:"slackDirectMessageUrl,omitempty"`
	Hidden                bool    `json:"hidden,omitempty"`
	ReadOnly              bool    `json:"readOnly,omitempty"`
	Source                *Source `json:"source,omitempty"`
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
