package eventcatalog

// Frontmatter types map exactly to EventCatalog's expected YAML frontmatter.
// Each struct marshals to YAML via go-faster/yaml, replacing the manual
// frontmatterWriter. Field declaration order = YAML output order.

type pointer struct {
	ID      string `yaml:"id"`
	Version string `yaml:"version,omitempty"`
}

type badgeFM struct {
	Content         string `yaml:"content"`
	BackgroundColor string `yaml:"backgroundColor,omitempty"`
	TextColor       string `yaml:"textColor,omitempty"`
	Icon            string `yaml:"icon,omitempty"`
	URL             string `yaml:"url,omitempty"`
}

type repositoryFM struct {
	Language string `yaml:"language,omitempty"`
	URL      string `yaml:"url,omitempty"`
}

type operationFM struct {
	Method      string   `yaml:"method"`
	Path        string   `yaml:"path"`
	StatusCodes []string `yaml:"statusCodes,omitempty,flow"`
}

type specificationFM struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	Name string `yaml:"name,omitempty"`
}

type attachmentFM struct {
	URL         string `yaml:"url"`
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type,omitempty"`
	Icon        string `yaml:"icon,omitempty"`
}

type changeFM struct {
	Version string `yaml:"version"`
	Date    string `yaml:"date,omitempty"`
	Summary string `yaml:"summary"`
}

type ubiquitousLanguageTermFM struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type agentModelFM struct {
	Provider string `yaml:"provider"`
	Name     string `yaml:"name"`
	Version  string `yaml:"version,omitempty"`
}

type agentToolFM struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type,omitempty"`
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description,omitempty"`
	Icon        string `yaml:"icon,omitempty"`
}

// --- Resource frontmatter types ---

type messageFM struct {
	ID         string             `yaml:"id"`
	Name       string             `yaml:"name"`
	Version    string             `yaml:"version"`
	Summary    string             `yaml:"summary,omitempty"`
	Deprecated bool               `yaml:"deprecated,omitempty"`
	Owners     []string           `yaml:"owners,omitempty"`
	Labels     map[string]string  `yaml:"labels,omitempty"`
	Changelog  []changeFM         `yaml:"changelog,omitempty"`
	Producers  []pointer          `yaml:"producers,omitempty"`
	Consumers  []pointer          `yaml:"consumers,omitempty"`
	Operation  *operationFM       `yaml:"operation,omitempty"`
	Badges     []badgeFM          `yaml:"badges,omitempty"`
	Repository *repositoryFM      `yaml:"repository,omitempty"`
	SchemaPath string             `yaml:"schemaPath,omitempty"`
}

type serviceFM struct {
	ID             string            `yaml:"id"`
	Name           string            `yaml:"name"`
	Version        string            `yaml:"version"`
	Summary        string            `yaml:"summary,omitempty"`
	Owners         []string          `yaml:"owners,omitempty"`
	Sends          []pointer         `yaml:"sends,omitempty"`
	Receives       []pointer         `yaml:"receives,omitempty"`
	WritesTo       []pointer         `yaml:"writesTo,omitempty"`
	ReadsFrom      []pointer         `yaml:"readsFrom,omitempty"`
	Entities       []string          `yaml:"entities,omitempty,flow"`
	Flows          []string          `yaml:"flows,omitempty,flow"`
	ExternalSystem bool              `yaml:"externalSystem,omitempty"`
	Badges         []badgeFM         `yaml:"badges,omitempty"`
	Repository     *repositoryFM     `yaml:"repository,omitempty"`
	Specifications []specificationFM `yaml:"specifications,omitempty"`
	Attachments    []attachmentFM    `yaml:"attachments,omitempty"`
}

type domainFM struct {
	ID                 string                       `yaml:"id"`
	Name               string                       `yaml:"name"`
	Version            string                       `yaml:"version"`
	Summary            string                       `yaml:"summary,omitempty"`
	Owners             []string                     `yaml:"owners,omitempty"`
	Services           []pointer                    `yaml:"services,omitempty"`
	Sends              []pointer                    `yaml:"sends,omitempty"`
	Receives           []pointer                    `yaml:"receives,omitempty"`
	Entities           []string                     `yaml:"entities,omitempty,flow"`
	Flows              []string                     `yaml:"flows,omitempty,flow"`
	Domains            []string                     `yaml:"domains,omitempty,flow"`
	DataProducts       []string                     `yaml:"data-products,omitempty,flow"`
	UbiquitousLanguage []ubiquitousLanguageTermFM   `yaml:"ubiquitousLanguage,omitempty"`
	Badges             []badgeFM                    `yaml:"badges,omitempty"`
	Attachments        []attachmentFM               `yaml:"attachments,omitempty"`
}

type entityFM struct {
	ID         string    `yaml:"id"`
	Name       string    `yaml:"name"`
	Version    string    `yaml:"version"`
	Summary    string    `yaml:"summary,omitempty"`
	Owners     []string  `yaml:"owners,omitempty"`
	Badges     []badgeFM `yaml:"badges,omitempty"`
	SchemaPath string    `yaml:"schemaPath,omitempty"`
}

type dataProductFM struct {
	ID      string    `yaml:"id"`
	Name    string    `yaml:"name"`
	Version string    `yaml:"version"`
	Summary string    `yaml:"summary,omitempty"`
	Owners  []string  `yaml:"owners,omitempty"`
	Inputs  []pointer `yaml:"inputs,omitempty"`
	Outputs []pointer `yaml:"outputs,omitempty"`
	Badges  []badgeFM `yaml:"badges,omitempty"`
}

type agentFM struct {
	ID        string        `yaml:"id"`
	Name      string        `yaml:"name"`
	Version   string        `yaml:"version"`
	Summary   string        `yaml:"summary,omitempty"`
	Owners    []string      `yaml:"owners,omitempty"`
	Sends     []pointer     `yaml:"sends,omitempty"`
	Receives  []pointer     `yaml:"receives,omitempty"`
	ReadsFrom []pointer     `yaml:"readsFrom,omitempty"`
	WritesTo  []pointer     `yaml:"writesTo,omitempty"`
	Model     *agentModelFM `yaml:"model,omitempty"`
	Tools     []agentToolFM `yaml:"tools,omitempty"`
	Flows     []string      `yaml:"flows,omitempty,flow"`
	Badges    []badgeFM     `yaml:"badges,omitempty"`
}

type channelParamFM struct {
	Enum        []string `yaml:"enum,omitempty,flow"`
	Default     string   `yaml:"default,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

type channelRouteFM struct {
	ID string `yaml:"id"`
}

type channelFM struct {
	ID                string                   `yaml:"id"`
	Name              string                   `yaml:"name"`
	Version           string                   `yaml:"version"`
	Summary           string                   `yaml:"summary,omitempty"`
	Address           string                   `yaml:"address,omitempty"`
	Protocols         []string                 `yaml:"protocols,omitempty,flow"`
	Messages          []pointer                `yaml:"messages,omitempty"`
	DeliveryGuarantee string                   `yaml:"deliveryGuarantee,omitempty"`
	Parameters        map[string]channelParamFM `yaml:"parameters,omitempty"`
	Routes            []channelRouteFM         `yaml:"routes,omitempty"`
	Owners            []string                 `yaml:"owners,omitempty"`
	Badges            []badgeFM                `yaml:"badges,omitempty"`
}

type dataStoreFM struct {
	ID             string    `yaml:"id"`
	Name           string    `yaml:"name"`
	Version        string    `yaml:"version"`
	ContainerType  string    `yaml:"container_type"`
	Summary        string    `yaml:"summary,omitempty"`
	Technology     string    `yaml:"technology,omitempty"`
	Classification string    `yaml:"classification,omitempty"`
	Retention      string    `yaml:"retention,omitempty"`
	Residency      string    `yaml:"residency,omitempty"`
	Owners         []string  `yaml:"owners,omitempty"`
	Badges         []badgeFM `yaml:"badges,omitempty"`
}

type flowEdgeFM struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label,omitempty"`
}

type flowStepFM struct {
	ID          string      `yaml:"id"`
	Title       string      `yaml:"title"`
	Summary     string      `yaml:"summary,omitempty"`
	Service     *pointer    `yaml:"service,omitempty"`
	Message     *pointer    `yaml:"message,omitempty"`
	Channel     *pointer    `yaml:"channel,omitempty"`
	Actor       *flowActor  `yaml:"actor,omitempty"`
	ExternalSys *flowActor  `yaml:"externalSystem,omitempty"`
	Custom      *flowCustom `yaml:"custom,omitempty"`
	Agent       *pointer    `yaml:"agent,omitempty"`
	DataStore   *pointer    `yaml:"dataStore,omitempty"`
	DataProduct *pointer    `yaml:"dataProduct,omitempty"`
	SubFlow     *pointer    `yaml:"subFlow,omitempty"`
	NextStep    *flowEdgeFM `yaml:"next_step,omitempty"`
	NextSteps   []flowEdgeFM `yaml:"next_steps,omitempty"`
}

type flowActor struct {
	Name    string `yaml:"name"`
	Summary string `yaml:"summary,omitempty"`
	URL     string `yaml:"url,omitempty"`
}

type flowCustom struct {
	Title   string `yaml:"title"`
	Icon    string `yaml:"icon,omitempty"`
	Type    string `yaml:"type,omitempty"`
	Summary string `yaml:"summary,omitempty"`
	URL     string `yaml:"url,omitempty"`
	Color   string `yaml:"color,omitempty"`
}

type flowFM struct {
	ID      string      `yaml:"id"`
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
	Summary string      `yaml:"summary,omitempty"`
	Badges  []badgeFM   `yaml:"badges,omitempty"`
	Steps   []flowStepFM `yaml:"steps,omitempty"`
}

type teamFM struct {
	ID                    string `yaml:"id"`
	Name                  string `yaml:"name"`
	Summary               string `yaml:"summary,omitempty"`
	Members               []string `yaml:"members,omitempty,flow"`
	Email                 string `yaml:"email,omitempty"`
	SlackDirectMessageURL string `yaml:"slackDirectMessageUrl,omitempty"`
}

type userFM struct {
	ID                    string `yaml:"id"`
	Name                  string `yaml:"name"`
	AvatarURL             string `yaml:"avatarUrl,omitempty"`
	Role                  string `yaml:"role,omitempty"`
	Email                 string `yaml:"email,omitempty"`
	SlackDirectMessageURL string `yaml:"slackDirectMessageUrl,omitempty"`
}
