package docserver

import (
	"net/http"

	"github.com/a-h/templ"
)

// renderComponent writes a templ component as an HTML response. Rendering
// errors mid-stream are best-effort: templ streams directly to the writer,
// so a 500 can only be sent if nothing was written yet.
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render documentation page", http.StatusInternalServerError)
	}
}
