package eventcatalog

// Extended resource frontmatter types for EventCatalog.
// Split from frontmatter_types.go to stay under the 350-line limit.

type channelParamFM struct {
	Enum        []string `yaml:"enum,omitempty,flow"`
	Default     string   `yaml:"default,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

type channelRouteFM struct {
	ID string `yaml:"id"`
}

// channelMessageFM points at a message from channel frontmatter.
// EventCatalog requires collection ("events"|"commands"|"queries"), the
// message's display name, and the id/version pointer — bare {id, version}
// pointers fail schema validation with "collection: Required".
type channelMessageFM struct {
	Collection string `yaml:"collection"`
	Name       string `yaml:"name"`
	ID         string `yaml:"id"`
	Version    string `yaml:"version"`
}

type channelFM struct {
	ID                string                    `yaml:"id"`
	Name              string                    `yaml:"name"`
	Version           string                    `yaml:"version"`
	Summary           string                    `yaml:"summary,omitempty"`
	Address           string                    `yaml:"address,omitempty"`
	Protocols         []string                  `yaml:"protocols,omitempty,flow"`
	Messages          []channelMessageFM        `yaml:"messages,omitempty"`
	DeliveryGuarantee string                    `yaml:"deliveryGuarantee,omitempty"`
	Parameters        map[string]channelParamFM `yaml:"parameters,omitempty"`
	Routes            []channelRouteFM          `yaml:"routes,omitempty"`
	Owners            []string                  `yaml:"owners,omitempty"`
	Badges            []badgeFM                 `yaml:"badges,omitempty"`
}

type dataStoreFM struct {
	ID             string    `yaml:"id"`
	Name           string    `yaml:"name"`
	Version        string    `yaml:"version"`
	ContainerType  string    `yaml:"container_type"` //nolint:tagliatelle // EventCatalog format
	Summary        string    `yaml:"summary,omitempty"`
	Technology     string    `yaml:"technology,omitempty"`
	Classification string    `yaml:"classification,omitempty"`
	Retention      string    `yaml:"retention,omitempty"`
	Residency      string    `yaml:"residency,omitempty"`
	Authoritative  bool      `yaml:"authoritative,omitempty"`
	AccessMode     string    `yaml:"access_mode,omitempty"` //nolint:tagliatelle // EventCatalog format
	Owners         []string  `yaml:"owners,omitempty"`
	Badges         []badgeFM `yaml:"badges,omitempty"`
}

type flowEdgeFM struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label,omitempty"`
}

type flowStepFM struct {
	ID          string       `yaml:"id"`
	Title       string       `yaml:"title"`
	Summary     string       `yaml:"summary,omitempty"`
	Service     *pointer     `yaml:"service,omitempty"`
	Message     *pointer     `yaml:"message,omitempty"`
	Actor       *flowActor   `yaml:"actor,omitempty"`
	ExternalSys *flowActor   `yaml:"externalSystem,omitempty"`
	Custom      *flowCustom  `yaml:"custom,omitempty"`
	Agent       *pointer     `yaml:"agent,omitempty"`
	Container   *pointer     `yaml:"container,omitempty"`
	DataProduct *pointer     `yaml:"dataProduct,omitempty"`
	Flow        *pointer     `yaml:"flow,omitempty"`
	NextStep    *flowEdgeFM  `yaml:"next_step,omitempty"`  //nolint:tagliatelle // EventCatalog format
	NextSteps   []flowEdgeFM `yaml:"next_steps,omitempty"` //nolint:tagliatelle // EventCatalog format
}

// flowActor matches EventCatalog's step actor shape: name and summary only.
// (externalSystem additionally allows url; plain actors do not.)
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
	ID      string       `yaml:"id"`
	Name    string       `yaml:"name"`
	Version string       `yaml:"version"`
	Summary string       `yaml:"summary,omitempty"`
	Owners  []string     `yaml:"owners,omitempty"`
	Badges  []badgeFM    `yaml:"badges,omitempty"`
	Steps   []flowStepFM `yaml:"steps,omitempty"`
}

type sourceFM struct {
	Provider string `yaml:"provider"`
	ID       string `yaml:"id,omitempty"`
	URL      string `yaml:"url,omitempty"`
}

type teamFM struct {
	ID                    string    `yaml:"id"`
	Name                  string    `yaml:"name"`
	Summary               string    `yaml:"summary,omitempty"`
	Members               []string  `yaml:"members,omitempty,flow"`
	Email                 string    `yaml:"email,omitempty"`
	SlackDirectMessageURL string    `yaml:"slackDirectMessageUrl,omitempty"`
	Hidden                bool      `yaml:"hidden,omitempty"`
	ReadOnly              bool      `yaml:"readOnly,omitempty"`
	Source                *sourceFM `yaml:"source,omitempty"`

	// EventCatalog teams have no role/avatarUrl fields (users do); both ride
	// the sanctioned x- custom-property escape hatch.
	XRole      string `yaml:"x-role,omitempty"`      //nolint:tagliatelle // custom property
	XAvatarURL string `yaml:"x-avatarUrl,omitempty"` //nolint:tagliatelle // custom property
}

type userFM struct {
	ID                    string    `yaml:"id"`
	Name                  string    `yaml:"name"`
	AvatarURL             string    `yaml:"avatarUrl,omitempty"`
	Role                  string    `yaml:"role,omitempty"`
	Email                 string    `yaml:"email,omitempty"`
	SlackDirectMessageURL string    `yaml:"slackDirectMessageUrl,omitempty"`
	Hidden                bool      `yaml:"hidden,omitempty"`
	ReadOnly              bool      `yaml:"readOnly,omitempty"`
	Source                *sourceFM `yaml:"source,omitempty"`
}

// customDocFM matches EventCatalog's customPages schema, where title and
// summary are REQUIRED (non-optional strings) and there is no id field.
type customDocFM struct {
	Title   string    `yaml:"title"`
	Summary string    `yaml:"summary"`
	Slug    string    `yaml:"slug,omitempty"`
	Owners  []string  `yaml:"owners,omitempty"`
	Badges  []badgeFM `yaml:"badges,omitempty"`
}

type schemaPointerFM struct {
	ID      string `yaml:"id,omitempty"`
	Ref     string `yaml:"$ref,omitempty"`
	File    string `yaml:"file,omitempty"`
	Path    string `yaml:"path,omitempty"`
	Name    string `yaml:"name,omitempty"`
	Format  string `yaml:"format,omitempty"`
	Default bool   `yaml:"default,omitempty"`
}

type sidebarFM struct {
	Badge string `yaml:"badge,omitempty"`
	Label string `yaml:"label,omitempty"`
}

type nodeStylesFM struct {
	Color string `yaml:"color,omitempty"`
	Label string `yaml:"label,omitempty"`
}

type stylesFM struct {
	Icon string        `yaml:"icon,omitempty"`
	Node *nodeStylesFM `yaml:"node,omitempty"`
}

type draftFM struct {
	Title   string `yaml:"title,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type baseConfigFM struct {
	Sidebar    *sidebarFM `yaml:"sidebar,omitempty"`
	Styles     *stylesFM  `yaml:"styles,omitempty"`
	EditUrl    string     `yaml:"editUrl,omitempty"`
	Draft      *draftFM   `yaml:"draft,omitempty"`
	Visualiser *bool      `yaml:"visualiser,omitempty"`
}
