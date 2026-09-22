package eventcatalog

// Frontmatter types map exactly to EventCatalog's expected YAML frontmatter.
// Each struct marshals to YAML via go-faster/yaml, replacing the manual
// frontmatterWriter. Field declaration order = YAML output order.

type pointer struct {
	ID      string `yaml:"id"`
	Version string `yaml:"version,omitempty"`
}

// channelRefFM references a channel from message frontmatter
// (EventCatalog channelPointer: {id, version?, parameters?}).
type channelRefFM struct {
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

type responseFM struct {
	StatusCode  string `yaml:"statusCode"`
	Description string `yaml:"description,omitempty"`
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

type deprecationFM struct {
	Date    string `yaml:"date,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type messageFM struct {
	ID         string            `yaml:"id"`
	Name       string            `yaml:"name"`
	Version    string            `yaml:"version"`
	Summary    string            `yaml:"summary,omitempty"`
	Deprecated any               `yaml:"deprecated,omitempty"`
	Owners     []string          `yaml:"owners,omitempty"`
	Channels   []channelRefFM    `yaml:"channels,omitempty"`
	Schemas    []schemaPointerFM `yaml:"schemas,omitempty"`
	Producers  []string          `yaml:"producers,omitempty"`
	Consumers  []string          `yaml:"consumers,omitempty"`
	Operation  *operationFM      `yaml:"operation,omitempty"`
	Badges     []badgeFM         `yaml:"badges,omitempty"`
	Repository *repositoryFM     `yaml:"repository,omitempty"`
	SchemaPath string            `yaml:"schemaPath,omitempty"`

	// EventCatalog has no native fields for labels or REST responses;
	// both ride the sanctioned x- custom-property escape hatch instead
	// (withExtensionProperties allows any x-* key).
	XLabels    map[string]string `yaml:"x-labels,omitempty"`     //nolint:tagliatelle // custom property
	XResponses []responseFM      `yaml:"x-responses,omitempty"` //nolint:tagliatelle // custom property
}

type serviceFM struct {
	baseConfigFM `yaml:",inline"`

	ID             string            `yaml:"id"`
	Name           string            `yaml:"name"`
	Version        string            `yaml:"version"`
	Summary        string            `yaml:"summary,omitempty"`
	Owners         []string          `yaml:"owners,omitempty"`
	Sends          []pointer         `yaml:"sends,omitempty"`
	Receives       []pointer         `yaml:"receives,omitempty"`
	WritesTo       []pointer         `yaml:"writesTo,omitempty"`
	ReadsFrom      []pointer         `yaml:"readsFrom,omitempty"`
	Entities       []pointer         `yaml:"entities,omitempty"`
	Flows          []pointer         `yaml:"flows,omitempty"`
	ExternalSystem bool              `yaml:"externalSystem,omitempty"`
	Badges         []badgeFM         `yaml:"badges,omitempty"`
	Repository     *repositoryFM     `yaml:"repository,omitempty"`
	Specifications []specificationFM `yaml:"specifications,omitempty"`
	Attachments    []attachmentFM    `yaml:"attachments,omitempty"`
}

type domainFM struct {
	baseConfigFM `yaml:",inline"`

	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	Version      string         `yaml:"version"`
	Summary      string         `yaml:"summary,omitempty"`
	Owners       []string       `yaml:"owners,omitempty"`
	Services     []pointer      `yaml:"services,omitempty"`
	Sends        []pointer      `yaml:"sends,omitempty"`
	Receives     []pointer      `yaml:"receives,omitempty"`
	Entities     []pointer      `yaml:"entities,omitempty"`
	Flows        []pointer      `yaml:"flows,omitempty"`
	Domains      []pointer      `yaml:"domains,omitempty"`
	DataProducts []pointer      `yaml:"data-products,omitempty"` //nolint:tagliatelle // EventCatalog format
	Badges       []badgeFM      `yaml:"badges,omitempty"`
	Attachments  []attachmentFM `yaml:"attachments,omitempty"`
}

type entityPropertyFM struct {
	Name                 string `yaml:"name"`
	Type                 string `yaml:"type"`
	Required             bool   `yaml:"required,omitempty"`
	Description          string `yaml:"description,omitempty"`
	References           string `yaml:"references,omitempty"`
	ReferencesIdentifier string `yaml:"referencesIdentifier,omitempty"`
	RelationType         string `yaml:"relationType,omitempty"`
}

type entityFM struct {
	ID            string             `yaml:"id"`
	Name          string             `yaml:"name"`
	Version       string             `yaml:"version"`
	Summary       string             `yaml:"summary,omitempty"`
	AggregateRoot bool               `yaml:"aggregateRoot,omitempty"`
	Identifier    string             `yaml:"identifier,omitempty"`
	Properties    []entityPropertyFM `yaml:"properties,omitempty"`
	Owners        []string           `yaml:"owners,omitempty"`
	Badges        []badgeFM          `yaml:"badges,omitempty"`
	SchemaPath    string             `yaml:"schemaPath,omitempty"`
}

type dataContractFM struct {
	Path string `yaml:"path"`
	Name string `yaml:"name,omitempty"`
	Type string `yaml:"type,omitempty"`
}

type dataProductOutputFM struct {
	ID       string          `yaml:"id"`
	Version  string          `yaml:"version,omitempty"`
	Contract *dataContractFM `yaml:"contract,omitempty"`
}

type dataProductFM struct {
	ID      string                `yaml:"id"`
	Name    string                `yaml:"name"`
	Version string                `yaml:"version"`
	Summary string                `yaml:"summary,omitempty"`
	Hidden  bool                  `yaml:"hidden,omitempty"`
	Owners  []string              `yaml:"owners,omitempty"`
	Inputs  []pointer             `yaml:"inputs,omitempty"`
	Outputs []dataProductOutputFM `yaml:"outputs,omitempty"`
	Badges  []badgeFM             `yaml:"badges,omitempty"`
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
	Flows     []pointer     `yaml:"flows,omitempty"`
	Badges    []badgeFM     `yaml:"badges,omitempty"`
}

// dictionaryTermFM is an entry of a domain's ubiquitous-language.mdx
// dictionary file (EventCatalog loads it from a separate file, not frontmatter).
type dictionaryTermFM struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}
