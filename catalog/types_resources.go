package catalog

// DataStore represents a data store (database, cache, object store, etc.).
type DataStore struct {
	ID             DataStoreID      `json:"id"`
	Name           Name             `json:"name"`
	Version        Version          `json:"version"`
	Summary        Summary          `json:"summary,omitempty"`
	ContainerType  string           `json:"containerType"`
	Technology     string           `json:"technology,omitempty"`
	Classification string           `json:"classification,omitempty"`
	Retention      string           `json:"retention,omitempty"`
	Residency      string           `json:"residency,omitempty"`
	Owners         []string         `json:"owners,omitempty"`
	Badges         []Badge          `json:"badges,omitempty"`
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
// Exactly one of Service, Message, Channel, Actor, ExternalSystem, or Custom should be set.
type FlowStep struct {
	ID        string          `json:"id"`
	Title     Title           `json:"title"`
	Summary   Summary         `json:"summary,omitempty"`
	Service   *FlowStepRef    `json:"service,omitempty"`
	Message   *FlowStepRef    `json:"message,omitempty"`
	Channel   *FlowStepRef    `json:"channel,omitempty"`
	Actor     *FlowActor      `json:"actor,omitempty"`
	External  *FlowActor      `json:"external,omitempty"`
	Custom    *FlowCustomNode `json:"custom,omitempty"`
	NextStep  *FlowEdge       `json:"nextStep,omitempty"`
	NextSteps []FlowEdge      `json:"nextSteps,omitempty"`
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
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
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
