package catalog

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
