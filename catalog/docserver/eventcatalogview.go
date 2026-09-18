package docserver

import (
	"encoding/json"
	"encoding/json/jsontext"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// This file builds the view models for the embedded event catalog
// (/docs/eventcatalog): a browsable, EventCatalog-style inventory of the
// services, messages, and channels in the catalog. All pages are
// server-rendered — no JavaScript required.

// eventCatalogMessageRow is one row in the message tables: a link to the
// message detail page plus the summary fields shown inline.
type eventCatalogMessageRow struct {
	Kind      string
	Name      string
	ID        string
	Href      string
	Summary   string
	Producer  string
	Consumers string
}

// eventCatalogProperty is one row of the schema property table on the
// message detail page.
type eventCatalogProperty struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Details     string
}

// eventCatalogMessageDetail is the view model of the message detail page.
type eventCatalogMessageDetail struct {
	Brand      string
	DocsPath   string
	CatalogRef string
	Kind       string
	Name       string
	ID         string
	Version    string
	Summary    string
	Deprecated bool
	Producers  []string
	Consumers  []string
	Channels   []eventCatalogLink
	Properties []eventCatalogProperty
	SchemaJSON string
	Examples   []string
	Changelog  []catalog.Change
}

// eventCatalogLink is a labeled link used in lists across the catalog pages.
type eventCatalogLink struct {
	Label string
	Href  string
}

// eventCatalogChannelRow is one channel entry on the overview page.
type eventCatalogChannelRow struct {
	Name       string
	Href       string
	Address    string
	Protocols  string
	Guarantee  string
	MessageCnt int
}

// eventCatalogServiceRow is one service entry on the overview page.
type eventCatalogServiceRow struct {
	Name     string
	Href     string
	ID       string
	Version  string
	Summary  string
	Commands int
	Events   int
	Queries  int
}

// eventCatalogOverview is the view model of the event catalog landing page.
type eventCatalogOverview struct {
	Brand      string
	DocsPath   string
	CatalogRef string
	Version    string
	Services   []eventCatalogServiceRow
	Channels   []eventCatalogChannelRow
	Messages   []eventCatalogMessageRow
}

// messageKindOrder ranks message kinds for stable table ordering.
func messageKindOrder(kind catalog.MessageKind) int {
	switch kind {
	case catalog.EventMessage:
		return 0
	case catalog.CommandMessage:
		return 1
	case catalog.QueryMessage:
		return 2
	default:
		return 3
	}
}

// serviceNames resolves service IDs to display names.
func serviceNames(cat *catalog.Catalog, ids []catalog.ServiceID) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, string(id))
	}

	for _, svc := range cat.Services {
		for i, id := range ids {
			if svc.ID == id && string(svc.Name) != "" {
				names[i] = string(svc.Name)
			}
		}
	}

	sort.Strings(names)

	return names
}

// collectMessages deduplicates every message in the catalog by key and
// returns them in stable kind-then-name order.
func collectMessages(cat *catalog.Catalog) []catalog.Message {
	seen := map[catalog.MessageID]bool{}
	messages := make([]catalog.Message, 0, 64)

	for _, svc := range cat.Services {
		for _, group := range [][]catalog.Message{svc.Commands, svc.Events, svc.Queries} {
			for _, msg := range group {
				if seen[catalog.Key(msg)] {
					continue
				}

				seen[catalog.Key(msg)] = true
				messages = append(messages, msg)
			}
		}
	}

	sort.Slice(messages, func(i, j int) bool {
		if ki, kj := messageKindOrder(messages[i].Kind), messageKindOrder(messages[j].Kind); ki != kj {
			return ki < kj
		}

		return catalog.Key(messages[i]) < catalog.Key(messages[j])
	})

	return messages
}

func findMessage(cat *catalog.Catalog, id string) (catalog.Message, bool) {
	for _, msg := range collectMessages(cat) {
		if string(catalog.Key(msg)) == id {
			return msg, true
		}
	}

	return catalog.Message{}, false
}

func findChannel(cat *catalog.Catalog, id string) (catalog.Channel, bool) {
	for _, ch := range cat.Channels {
		if string(ch.ID) == id {
			return ch, true
		}
	}

	return catalog.Channel{}, false
}

func findService(cat *catalog.Catalog, id string) (catalog.Service, bool) {
	for _, svc := range cat.Services {
		if string(svc.ID) == id {
			return svc, true
		}
	}

	return catalog.Service{}, false
}

// newEventCatalogOverview builds the landing-page view model.
func newEventCatalogOverview(cfg Config, cat *catalog.Catalog) eventCatalogOverview {
	serviceNamesByID := map[catalog.ServiceID]string{}
	for _, svc := range cat.Services {
		serviceNamesByID[svc.ID] = cmpOr(string(svc.Name), string(svc.ID))
	}

	rows := make([]eventCatalogMessageRow, 0, 64)
	for _, msg := range collectMessages(cat) {
		rows = append(rows, eventCatalogMessageRow{
			Kind:      string(msg.Kind),
			Name:      cmpOr(string(msg.Name), string(msg.ID)),
			ID:        string(catalog.Key(msg)),
			Href:      eventCatalogMessageHref(cfg.DocsPath, catalog.Key(msg)),
			Summary:   string(msg.Summary),
			Producer:  strings.Join(serviceNames(cat, msg.Producers), ", "),
			Consumers: strings.Join(serviceNames(cat, msg.Consumers), ", "),
		})
	}

	channels := make([]eventCatalogChannelRow, 0, len(cat.Channels))
	for _, ch := range cat.Channels {
		channels = append(channels, eventCatalogChannelRow{
			Name:       cmpOr(string(ch.Name), string(ch.ID)),
			Href:       eventCatalogChannelHref(cfg.DocsPath, string(ch.ID)),
			Address:    string(ch.Address),
			Protocols:  strings.Join(protocolsOf(ch), ", "),
			Guarantee:  string(ch.DeliveryGuarantee),
			MessageCnt: len(ch.Messages),
		})
	}

	services := make([]eventCatalogServiceRow, 0, len(cat.Services))
	for _, svc := range cat.Services {
		services = append(services, eventCatalogServiceRow{
			Name:     cmpOr(string(svc.Name), string(svc.ID)),
			Href:     eventCatalogServiceHref(cfg.DocsPath, string(svc.ID)),
			ID:       string(svc.ID),
			Version:  string(svc.Version),
			Summary:  string(svc.Summary),
			Commands: len(svc.Commands),
			Events:   len(svc.Events),
			Queries:  len(svc.Queries),
		})
	}

	return eventCatalogOverview{
		Brand:      cmpOr(cfg.ServiceName, string(cat.Title)),
		DocsPath:   cfg.DocsPath,
		CatalogRef: cfg.DocsPath + "/catalog.json",
		Version:    cmpOr(string(cat.Version), cfg.Version),
		Services:   services,
		Channels:   channels,
		Messages:   rows,
	}
}

// newEventCatalogMessageDetail builds the message detail view model. The
// second return value is false when no message with the given key exists.
func newEventCatalogMessageDetail(cfg Config, cat *catalog.Catalog, id string) (eventCatalogMessageDetail, bool) {
	msg, ok := findMessage(cat, id)
	if !ok {
		return eventCatalogMessageDetail{}, false
	}

	detail := eventCatalogMessageDetail{
		Brand:      cmpOr(cfg.ServiceName, string(cat.Title)),
		DocsPath:   cfg.DocsPath,
		CatalogRef: cfg.DocsPath + "/catalog.json",
		Kind:       string(msg.Kind),
		Name:       cmpOr(string(msg.Name), string(msg.ID)),
		ID:         string(catalog.Key(msg)),
		Version:    string(msg.Version),
		Summary:    string(msg.Summary),
		Deprecated: msg.Deprecated,
		Channels:   channelLinks(cat, cfg.DocsPath, msg.Channels),
		Producers:  serviceNames(cat, msg.Producers),
		Consumers:  serviceNames(cat, msg.Consumers),
		SchemaJSON: prettySchema(msg.Schema),
		Examples:   prettyExamples(msg.Examples),
		Changelog:  msg.Changelog,
	}

	if msg.Schema != nil {
		detail.Properties = schemaProperties(msg.Schema)
	}

	return detail, true
}

// channelLinks resolves channel IDs to labeled links toward the channel
// detail pages. Unknown channels fall back to a plain label.
func channelLinks(cat *catalog.Catalog, docsPath string, ids []catalog.ChannelID) []eventCatalogLink {
	links := make([]eventCatalogLink, 0, len(ids))

	for _, id := range ids {
		label := string(id)
		if ch, ok := findChannel(cat, string(id)); ok && ch.Name != "" {
			label = string(ch.Name)
		}

		links = append(links, eventCatalogLink{Label: label, Href: eventCatalogChannelHref(docsPath, string(id))})
	}

	return links
}

// schemaProperties flattens the top-level schema properties into table rows.
func schemaProperties(s *schema.Schema) []eventCatalogProperty {
	required := map[string]bool{}
	for _, name := range s.Required {
		required[name] = true
	}

	rows := make([]eventCatalogProperty, 0, len(s.Properties))
	for _, name := range sortedKeys(s.Properties) {
		prop := s.Properties[name]
		rows = append(rows, eventCatalogProperty{
			Name:        name,
			Type:        propertyTypeLabel(prop),
			Description: prop.Description,
			Required:    required[name],
			Details:     propertyDetails(prop),
		})
	}

	return rows
}

func propertyTypeLabel(p schema.Property) string {
	if p.Type == schema.TypeArray && p.Items != nil {
		return "array of " + string(p.Items.Type)
	}

	return string(p.Type)
}

// propertyDetails concatenates enum values, format, default, and nullability
// into a compact, human-readable hint line.
func propertyDetails(p schema.Property) string {
	parts := make([]string, 0, 4)
	if len(p.Enum) > 0 {
		parts = append(parts, "enum: "+strings.Join(p.Enum, " | "))
	}

	if p.Format != "" {
		parts = append(parts, "format: "+p.Format)
	}

	if p.Default != "" {
		parts = append(parts, "default: "+p.Default)
	}

	if p.Nullable {
		parts = append(parts, "nullable")
	}

	return strings.Join(parts, " · ")
}

// prettySchema renders the schema as indented JSON for the detail page.
func prettySchema(s *schema.Schema) string {
	if s == nil {
		return ""
	}

	b, err := json.Marshal(s, jsontext.WithIndent("  "))
	if err != nil {
		return ""
	}

	return string(b)
}

// prettyExamples renders each stored example payload as indented JSON.
func prettyExamples(examples []jsontext.Value) []string {
	out := make([]string, 0, len(examples))
	for _, ex := range examples {
		indented, err := ex.Indent("", "  ")
		if err != nil {
			continue
		}

		out = append(out, string(indented))
	}

	return out
}

func sortedKeys(m map[string]schema.Property) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func protocolsOf(ch catalog.Channel) []string {
	protocols := make([]string, 0, len(ch.Protocols))
	for _, p := range ch.Protocols {
		protocols = append(protocols, string(p))
	}

	return protocols
}

func eventCatalogHref(docsPath string) string {
	return docsPath + "/eventcatalog"
}

func eventCatalogMessageHref(docsPath string, id catalog.MessageID) string {
	return docsPath + "/eventcatalog/messages/" + url.PathEscape(string(id))
}

func eventCatalogChannelHref(docsPath, id string) string {
	return docsPath + "/eventcatalog/channels/" + url.PathEscape(id)
}

func eventCatalogServiceHref(docsPath, id string) string {
	return docsPath + "/eventcatalog/services/" + url.PathEscape(id)
}

// cmpOr returns a if non-empty, else b (local helper to avoid importing cmp
// for two call sites in the view builders).
func cmpOr(a, b string) string {
	if a != "" {
		return a
	}

	return b
}

// The DocsServer handlers below serve the event catalog pages.

func (ds *DocsServer) serveEventCatalog(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)
	ds.renderComponent(w, r, EventCatalogPage(newEventCatalogOverview(ds.config, ds.provider())))
}

func (ds *DocsServer) serveEventCatalogMessage(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	detail, ok := newEventCatalogMessageDetail(ds.config, ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponent(w, r, eventCatalogNotFound(ds.config.ServiceName, ds.config.DocsPath, "message", r.PathValue("id")))

		return
	}

	ds.renderComponent(w, r, EventCatalogMessagePage(detail))
}

func (ds *DocsServer) serveEventCatalogChannel(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	ch, ok := findChannel(ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponent(w, r, eventCatalogNotFound(ds.config.ServiceName, ds.config.DocsPath, "channel", r.PathValue("id")))

		return
	}

	ds.renderComponent(w, r, EventCatalogChannelPage(newEventCatalogChannelDetail(ds.config, ds.provider(), ch)))
}

func (ds *DocsServer) serveEventCatalogService(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	svc, ok := findService(ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponent(w, r, eventCatalogNotFound(ds.config.ServiceName, ds.config.DocsPath, "service", r.PathValue("id")))

		return
	}

	ds.renderComponent(w, r, EventCatalogServicePage(newEventCatalogServiceDetail(ds.config, svc)))
}
