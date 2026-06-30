package catalog

// UbiquitousLanguageTerm represents a single term in a domain's ubiquitous language (DDD glossary).
type UbiquitousLanguageTerm struct {
	Name        Name   `json:"name"`
	Description string `json:"description,omitempty"`
}

// Entity represents a first-class domain entity with optional schema, owners, and badges.
type Entity struct {
	ID      EntityID `json:"id"`
	Name    Name     `json:"name"`
	Version Version  `json:"version"`
	Summary Summary  `json:"summary,omitempty"`
	Schema  *Schema  `json:"schema,omitempty"`
	Owners  []string `json:"owners,omitempty"`
	Badges  []Badge  `json:"badges,omitempty"`
}

// DataProduct represents a data product in a data mesh — a curated, owned dataset.
type DataProduct struct {
	ID      DataProductID `json:"id"`
	Name    Name          `json:"name"`
	Version Version       `json:"version"`
	Summary Summary       `json:"summary,omitempty"`
	Domain  DomainID      `json:"domain,omitempty"`
	Schema  *Schema       `json:"schema,omitempty"`
	Owners  []string      `json:"owners,omitempty"`
	Badges  []Badge       `json:"badges,omitempty"`
}

// Agent represents an AI agent that can own messages, services, data stores, and flows.
type Agent struct {
	ID         AgentID       `json:"id"`
	Name       Name          `json:"name"`
	Version    Version       `json:"version"`
	Summary    Summary       `json:"summary,omitempty"`
	Commands   []Message     `json:"commands,omitempty"`
	Events     []Message     `json:"events,omitempty"`
	Queries    []Message     `json:"queries,omitempty"`
	DataStores []DataStoreID `json:"dataStores,omitempty"`
	Flows      []FlowID      `json:"flows,omitempty"`
	Owners     []string      `json:"owners,omitempty"`
	Badges     []Badge       `json:"badges,omitempty"`
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
