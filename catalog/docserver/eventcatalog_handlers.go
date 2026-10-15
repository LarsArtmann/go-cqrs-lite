package docserver

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

// The DocsServer handlers below serve the event catalog pages.

func (ds *DocsServer) serveEventCatalog(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)
	ds.renderComponent(w, r, EventCatalogPage(newEventCatalogOverview(ds.config, ds.provider())))
}

// serveEventCatalogErrors renders the error-surface page. The three states
// are explicit: no file configured (info alert), broken file (error alert
// with the cause), valid export (stats + table). An unreadable file must
// never render as an empty — silently clean — surface.
func (ds *DocsServer) serveEventCatalogErrors(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	var (
		flow    *eventcatalog.Errorflow
		readErr error
	)
	if path := ds.config.EventCatalogErrorsFile; path != "" {
		flow, readErr = eventcatalog.ReadErrorflow(path)
	}

	ds.renderComponent(
		w,
		r,
		EventCatalogErrorsPage(newEventCatalogErrorsDetail(ds.config, flow, readErr)),
	)
}

func (ds *DocsServer) serveEventCatalogMessage(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	detail, ok := newEventCatalogMessageDetail(ds.config, ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponentStatus(
			w,
			r,
			http.StatusNotFound,
			eventCatalogNotFound(
				ds.config.ServiceName,
				ds.config.DocsPath,
				"message",
				r.PathValue("id"),
			),
		)

		return
	}

	ds.renderComponent(w, r, EventCatalogMessagePage(detail))
}

func (ds *DocsServer) serveEventCatalogChannel(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	ch, ok := findChannel(ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponentStatus(
			w,
			r,
			http.StatusNotFound,
			eventCatalogNotFound(
				ds.config.ServiceName,
				ds.config.DocsPath,
				"channel",
				r.PathValue("id"),
			),
		)

		return
	}

	ds.renderComponent(
		w,
		r,
		EventCatalogChannelPage(newEventCatalogChannelDetail(ds.config, ds.provider(), ch)),
	)
}

func (ds *DocsServer) serveEventCatalogService(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	svc, ok := findService(ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponentStatus(
			w,
			r,
			http.StatusNotFound,
			eventCatalogNotFound(
				ds.config.ServiceName,
				ds.config.DocsPath,
				"service",
				r.PathValue("id"),
			),
		)

		return
	}

	ds.renderComponent(w, r, EventCatalogServicePage(newEventCatalogServiceDetail(ds.config, svc)))
}

func (ds *DocsServer) serveEventCatalogDataProduct(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	detail, ok := newEventCatalogDataProductDetail(ds.config, ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponentStatus(
			w,
			r,
			http.StatusNotFound,
			eventCatalogNotFound(
				ds.config.ServiceName,
				ds.config.DocsPath,
				"data product",
				r.PathValue("id"),
			),
		)

		return
	}

	ds.renderComponent(w, r, EventCatalogDataProductPage(detail))
}
