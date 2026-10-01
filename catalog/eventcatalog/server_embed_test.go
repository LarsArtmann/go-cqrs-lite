package eventcatalog_test

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

//go:embed testdata/dist
var embeddedDist embed.FS

// TestStaticServerServesEmbeddedFS proves the //go:embed -> fs.Sub -> server
// path compiles and serves a tree baked into the test binary, the deployment
// shape of an embedded EventCatalog.
func TestStaticServerServesEmbeddedFS(t *testing.T) {
	t.Parallel()

	sub, err := fs.Sub(embeddedDist, "testdata/dist")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}

	srv, err := eventcatalog.NewStaticServer(sub)
	if err != nil {
		t.Fatalf("NewStaticServer: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/foo/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Foo") {
		t.Fatalf("body = %q, want embedded Foo page", rec.Body.String())
	}
}
