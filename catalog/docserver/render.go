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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render documentation page", http.StatusInternalServerError)
	}
}
