package architecture

import "strings"

// resourceKind identifies the sort of catalog resource a section documents.
type resourceKind string

// The catalog's resource kinds, in reference-section order.
const (
	kindOverview    resourceKind = "overview"
	kindDomain      resourceKind = "domain"
	kindService     resourceKind = "service"
	kindCommand     resourceKind = "command"
	kindEvent       resourceKind = "event"
	kindQuery       resourceKind = "query"
	kindChannel     resourceKind = "channel"
	kindDataStore   resourceKind = "data-store"
	kindFlow        resourceKind = "flow"
	kindEntity      resourceKind = "entity"
	kindDataProduct resourceKind = "data-product"
	kindAgent       resourceKind = "agent"
	kindADR         resourceKind = "adr"
	kindTeam        resourceKind = "team"
	kindUser        resourceKind = "user"
	kindDoc         resourceKind = "doc"
)

// kindInfo describes how one resource kind appears in the document:
// which sidebar group it belongs to and how it is tagged and anchored.
type kindInfo struct {
	kind   resourceKind
	group  string // sidebar group heading
	label  string // singular name shown as the section eyebrow
	blurb  string // sentence under the group heading
	tag    string // short sidebar tag, e.g. "EVT"
	class  string // CSS modifier, e.g. "evt"
	prefix string // anchor prefix, e.g. "event"
}

// kindTable is the ordered registry of every rendered resource kind. The
// order here is the order of the sidebar and of the document body.
var kindTable = []kindInfo{ //nolint:gochecknoglobals // static vocabulary table, immutable by contract
	{
		kind:   kindOverview,
		group:  "Overview",
		label:  "Architecture reference",
		blurb:  "",
		tag:    "DOC",
		class:  "doc",
		prefix: "overview",
	},
	{
		kind:   kindDomain,
		group:  "Domains",
		label:  "Domain",
		blurb:  "Business areas and the services that belong to them.",
		tag:    "DOM",
		class:  "dom",
		prefix: "domain",
	},
	{
		kind:   kindService,
		group:  "Services",
		label:  "Service",
		blurb:  "Applications that send and receive messages.",
		tag:    "SVC",
		class:  "svc",
		prefix: "service",
	},
	{
		kind:   kindCommand,
		group:  "Commands",
		label:  "Command",
		blurb:  "Requests to change state.",
		tag:    "CMD",
		class:  "cmd",
		prefix: "command",
	},
	{
		kind:   kindEvent,
		group:  "Events",
		label:  "Event",
		blurb:  "Facts that have happened.",
		tag:    "EVT",
		class:  "evt",
		prefix: "event",
	},
	{
		kind:   kindQuery,
		group:  "Queries",
		label:  "Query",
		blurb:  "Requests for state.",
		tag:    "QRY",
		class:  "qry",
		prefix: "query",
	},
	{
		kind:   kindChannel,
		group:  "Channels",
		label:  "Channel",
		blurb:  "Where messages travel.",
		tag:    "CHN",
		class:  "chn",
		prefix: "channel",
	},
	{
		kind:   kindDataStore,
		group:  "Data stores",
		label:  "Data store",
		blurb:  "Where services keep their data.",
		tag:    "DB",
		class:  "dat",
		prefix: "data-store",
	},
	{
		kind:   kindFlow,
		group:  "Flows",
		label:  "Flow",
		blurb:  "Step-by-step business processes.",
		tag:    "FLOW",
		class:  "flw",
		prefix: "flow",
	},
	{
		kind:   kindEntity,
		group:  "Entities",
		label:  "Entity",
		blurb:  "The things the domain is about.",
		tag:    "ENT",
		class:  "ent",
		prefix: "entity",
	},
	{
		kind:   kindDataProduct,
		group:  "Data products",
		label:  "Data product",
		blurb:  "Published, contract-backed data.",
		tag:    "DP",
		class:  "dp",
		prefix: "data-product",
	},
	{
		kind:   kindAgent,
		group:  "Agents",
		label:  "Agent",
		blurb:  "Autonomous consumers and producers.",
		tag:    "AGT",
		class:  "agt",
		prefix: "agent",
	},
	{
		kind:   kindADR,
		group:  "Decisions",
		label:  "Decision record",
		blurb:  "Architecture decision records.",
		tag:    "ADR",
		class:  "adr",
		prefix: "adr",
	},
	{
		kind:   kindTeam,
		group:  "Teams",
		label:  "Team",
		blurb:  "Groups that own resources.",
		tag:    "TEAM",
		class:  "tm",
		prefix: "team",
	},
	{
		kind:   kindUser,
		group:  "People",
		label:  "Person",
		blurb:  "Individuals that own resources.",
		tag:    "USER",
		class:  "usr",
		prefix: "user",
	},
	{
		kind:   kindDoc,
		group:  "Documentation",
		label:  "Documentation",
		blurb:  "Additional pages from the catalog.",
		tag:    "DOC",
		class:  "doc",
		prefix: "doc",
	},
}

// kindInfos indexes kindTable by kind.
func kindInfos() map[resourceKind]kindInfo {
	index := make(map[resourceKind]kindInfo, len(kindTable))
	for _, info := range kindTable {
		index[info.kind] = info
	}

	return index
}

// anchor returns the in-document id for a resource of this kind. The
// document overview has no resource id and anchors to its prefix.
func (k kindInfo) anchor(id string) string {
	if id == "" {
		return k.prefix
	}

	return k.prefix + "-" + slug(id)
}

// slug turns a catalog identifier into an HTML id fragment: lowercase, with
// every character outside [a-z0-9_-] replaced by a dash.
func slug(id string) string {
	var b strings.Builder

	b.Grow(len(id))

	for _, r := range strings.ToLower(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}

	return b.String()
}
