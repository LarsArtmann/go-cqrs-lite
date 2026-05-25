package catalog

// DataStore represents a data store (database, cache, object store, etc.).
type DataStore struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Summary        string   `json:"summary,omitempty"`
	ContainerType  string   `json:"containerType"`
	Technology     string   `json:"technology,omitempty"`
	Classification string   `json:"classification,omitempty"`
	Retention      string   `json:"retention,omitempty"`
	Residency      string   `json:"residency,omitempty"`
	Owners         []string `json:"owners,omitempty"`
	Badges         []Badge  `json:"badges,omitempty"`
}

// Flow represents a message flow with ordered steps between services and messages.
type Flow struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Version string     `json:"version"`
	Summary string     `json:"summary,omitempty"`
	Steps   []FlowStep `json:"steps"`
	Badges  []Badge    `json:"badges,omitempty"`
}

// FlowStep represents a single step in a flow diagram.
// Exactly one of Service, Message, Channel, Actor, ExternalSystem, or Custom should be set.
type FlowStep struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Summary  string          `json:"summary,omitempty"`
	Service  *FlowStepRef    `json:"service,omitempty"`
	Message  *FlowStepRef    `json:"message,omitempty"`
	Channel  *FlowStepRef    `json:"channel,omitempty"`
	Actor    *FlowActor      `json:"actor,omitempty"`
	External *FlowActor      `json:"external,omitempty"`
	Custom   *FlowCustomNode `json:"custom,omitempty"`
	NextStep *FlowEdge       `json:"nextStep,omitempty"`
	NextSteps []FlowEdge     `json:"nextSteps,omitempty"`
}

// FlowStepRef references a catalog resource by ID and optional version.
type FlowStepRef struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

// FlowActor describes a person or external system in a flow step.
type FlowActor struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
	URL     string `json:"url,omitempty"`
}

// FlowCustomNode describes a custom node type in a flow step.
type FlowCustomNode struct {
	Title   string `json:"title"`
	Icon    string `json:"icon,omitempty"`
	Type    string `json:"type,omitempty"`
	Summary string `json:"summary,omitempty"`
	URL     string `json:"url,omitempty"`
	Color   string `json:"color,omitempty"`
}

// FlowEdge connects one step to the next with an optional label.
type FlowEdge struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// Team represents a team that owns catalog resources.
type Team struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Summary              string   `json:"summary,omitempty"`
	Members              []string `json:"members,omitempty"`
	Email                string   `json:"email,omitempty"`
	SlackDirectMessageURL string  `json:"slackDirectMessageUrl,omitempty"`
}

// User represents an individual who owns catalog resources.
type User struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	AvatarURL            string `json:"avatarUrl,omitempty"`
	Role                 string `json:"role,omitempty"`
	Email                string `json:"email,omitempty"`
	SlackDirectMessageURL string `json:"slackDirectMessageUrl,omitempty"`
}
