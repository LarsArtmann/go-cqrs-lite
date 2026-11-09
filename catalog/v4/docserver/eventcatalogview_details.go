package docserver

import (
	"sort"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// Detail-page view models for channels and services in the embedded event
// catalog. The message detail model lives in eventcatalogview.go.

// eventCatalogChannelDetail is the view model of the channel detail page.
type eventCatalogChannelDetail struct {
	Brand      string
	DocsPath   string
	CatalogRef string
	Name       string
	ID         string
	Version    string
	Summary    string
	Address    string
	Protocols  []string
	Guarantee  string
	Parameters map[string]catalog.ChannelParam
	Messages   []eventCatalogLink
}

// eventCatalogServiceDetail is the view model of the service detail page.
type eventCatalogServiceDetail struct {
	Brand       string
	DocsPath    string
	CatalogRef  string
	Name        string
	ID          string
	Version     string
	Summary     string
	Owners      []string
	External    bool
	WritesTo    []string
	ReadsFrom   []string
	Repository  string
	CommandRefs []eventCatalogLink
	EventRefs   []eventCatalogLink
	QueryRefs   []eventCatalogLink
}

func newEventCatalogChannelDetail(
	cfg Config,
	cat *catalog.Catalog,
	ch catalog.Channel,
) eventCatalogChannelDetail {
	links := make([]eventCatalogLink, 0, len(ch.Messages))
	for _, id := range ch.Messages {
		label := string(id)
		if msg, ok := findMessage(cat, string(id)); ok && msg.Name != "" {
			label = string(msg.Name)
		}

		links = append(
			links,
			eventCatalogLink{Label: label, Href: eventCatalogMessageHref(cfg.DocsPath, id)},
		)
	}

	sort.Slice(links, func(i, j int) bool { return links[i].Label < links[j].Label })

	return eventCatalogChannelDetail{
		Brand:      cmpOr(cfg.ServiceName, string(cat.Title)),
		DocsPath:   cfg.DocsPath,
		CatalogRef: cfg.DocsPath + "/catalog.json",
		Name:       cmpOr(string(ch.Name), string(ch.ID)),
		ID:         string(ch.ID),
		Version:    string(ch.Version),
		Summary:    string(ch.Summary),
		Address:    string(ch.Address),
		Protocols:  protocolsOf(ch),
		Guarantee:  string(ch.DeliveryGuarantee),
		Parameters: ch.Parameters,
		Messages:   links,
	}
}

func newEventCatalogServiceDetail(cfg Config, svc catalog.Service) eventCatalogServiceDetail {
	group := func(msgs []catalog.Message) []eventCatalogLink {
		links := make([]eventCatalogLink, 0, len(msgs))
		for _, msg := range msgs {
			links = append(links, eventCatalogLink{
				Label: cmpOr(string(msg.Name), string(msg.ID)),
				Href:  eventCatalogMessageHref(cfg.DocsPath, catalog.Key(msg)),
			})
		}

		sort.Slice(links, func(i, j int) bool { return links[i].Label < links[j].Label })

		return links
	}

	repo := ""
	if svc.Repository != nil {
		repo = string(svc.Repository.URL)
	}

	stores := func(ids []catalog.DataStoreID) []string {
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			names = append(names, string(id))
		}

		return names
	}

	return eventCatalogServiceDetail{
		Brand:       cmpOr(cfg.ServiceName, string(svc.Name)),
		DocsPath:    cfg.DocsPath,
		CatalogRef:  cfg.DocsPath + "/catalog.json",
		Name:        cmpOr(string(svc.Name), string(svc.ID)),
		ID:          string(svc.ID),
		Version:     string(svc.Version),
		Summary:     string(svc.Summary),
		Owners:      svc.Owners,
		External:    svc.ExternalSystem,
		WritesTo:    stores(svc.WritesTo),
		ReadsFrom:   stores(svc.ReadsFrom),
		Repository:  repo,
		CommandRefs: group(svc.Commands),
		EventRefs:   group(svc.Events),
		QueryRefs:   group(svc.Queries),
	}
}

// joinOr returns the comma-joined strings, or a fallback dash when empty.
func joinOr(values []string) string {
	if len(values) == 0 {
		return "—"
	}

	return strings.Join(values, ", ")
}
