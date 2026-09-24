package docserver

import "net/http"

// The DocsServer handlers below serve the event catalog pages.

func (ds *DocsServer) serveEventCatalog(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)
	ds.renderComponent(w, r, EventCatalogPage(newEventCatalogOverview(ds.config, ds.provider())))
}

func (ds *DocsServer) serveEventCatalogMessage(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)

	detail, ok := newEventCatalogMessageDetail(ds.config, ds.provider(), r.PathValue("id"))
	if !ok {
		ds.renderComponent(
			w,
			r,
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
		ds.renderComponent(
			w,
			r,
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
		ds.renderComponent(
			w,
			r,
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
		ds.renderComponent(
			w,
			r,
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
