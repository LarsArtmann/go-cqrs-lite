package docserver

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
)

// renderComponent writes a templ component as an HTML response. The
// component renders into an in-memory buffer first, then lands on the wire
// in one write: render errors surface as a clean 500 (nothing was sent
// yet), and the route-level ETag middleware can hash the page for
// conditional requests — templ's runtime flushes at the end of a render,
// which would otherwise switch the middleware to uncacheable streaming
// mode. Docs pages are bounded (kilobytes), so the buffer is cheap.
func (ds *DocsServer) renderComponent(
	w http.ResponseWriter,
	r *http.Request,
	component templ.Component,
) {
	ds.renderComponentStatus(w, r, http.StatusOK, component)
}

// renderComponentStatus writes a templ component with an explicit HTTP
// status. Not-found pages use it so the styled body does not soften a 404
// into a 200.
func (ds *DocsServer) renderComponentStatus(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	component templ.Component,
) {
	var page bytes.Buffer

	if err := component.Render(r.Context(), &page); err != nil {
		http.Error(w, "failed to render documentation page", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	_, _ = w.Write(page.Bytes())
}
