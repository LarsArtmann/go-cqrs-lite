package eventcatalog_test

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:embed testdata/site
var embeddedDist embed.FS

const (
	homeMarker     = "Home"
	fooMarker      = "Foo"
	legacyMarker   = "Legacy"
	notFoundMarker = "Not Found"
)

// staticSiteFS models the shape EventCatalog's `npx eventcatalog build` emits:
// an HTML index per route, extensionless legacy pages, JSON under api-style
// directories, and content-hashed bundles under _astro/.
func staticSiteFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":          {Data: []byte("<!doctype html><title>Home</title>")},
		"404.html":            {Data: []byte("<!doctype html><title>Not Found</title>")},
		"legacy.html":         {Data: []byte("<!doctype html><title>Legacy</title>")},
		"docs/foo/index.html": {Data: []byte("<!doctype html><title>Foo</title>")},
		"docs/data.json":      {Data: []byte(`{"ok":true}`)},
		"_astro/app.js":       {Data: []byte(`console.log("app")`)},
	}
}

var _ = Describe("Embedded EventCatalog server", func() {
	var srv *eventcatalog.StaticServer

	BeforeEach(func() {
		var err error
		srv, err = eventcatalog.NewStaticServer(staticSiteFS())
		Expect(err).ToNot(HaveOccurred())
	})

	// request drives the server exactly as an HTTP client would, without
	// reaching into the server's unexported routing helpers.
	request := func(method, target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(method, target, nil))

		return rec
	}

	Describe("As a developer embedding a built EventCatalog site", func() {
		Describe("constructing the server", func() {
			It("rejects a nil filesystem", func() {
				_, err := eventcatalog.NewStaticServer(nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("routing GET requests to the built site", func() {
			DescribeTable(
				"resolving a request path",
				func(target string, wantStatus int, wantBody, wantLocation string) {
					rec := request(http.MethodGet, target)

					Expect(rec.Code).To(Equal(wantStatus))
					if wantBody != "" {
						Expect(rec.Body.String()).To(ContainSubstring(wantBody))
					}
					Expect(rec.Header().Get("Location")).To(Equal(wantLocation))
				},
				Entry("serves the root index", "/", http.StatusOK, homeMarker, ""),
				Entry("serves an explicit index", "/index.html", http.StatusOK, homeMarker, ""),
				Entry("serves a directory index", "/docs/foo/", http.StatusOK, fooMarker, ""),
				Entry("serves an extensionless page", "/legacy", http.StatusOK, legacyMarker, ""),
				Entry(
					"serves an explicit html page",
					"/legacy.html",
					http.StatusOK,
					legacyMarker,
					"",
				),
				Entry(
					"redirects a slash-less directory to its canonical URL",
					"/docs/foo", http.StatusMovedPermanently, "", "/docs/foo/",
				),
				Entry(
					"serves the custom 404 for a missing route",
					"/missing", http.StatusNotFound, notFoundMarker, "",
				),
				Entry(
					"404s a directory without an index",
					"/docs", http.StatusNotFound, notFoundMarker, "",
				),
				Entry(
					"rejects path traversal",
					"/docs/../404.html", http.StatusNotFound, notFoundMarker, "",
				),
			)
		})

		Describe("caching and content types", func() {
			It("marks html responses no-cache", func() {
				rec := request(http.MethodGet, "/")

				Expect(rec.Header().Get("Cache-Control")).To(Equal("no-cache"))
				Expect(rec.Header().Get("Content-Type")).To(ContainSubstring("text/html"))
			})

			It("marks content-hashed astro bundles immutable", func() {
				rec := request(http.MethodGet, "/_astro/app.js")

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Header().Get("Cache-Control")).To(ContainSubstring("immutable"))
				Expect(rec.Header().Get("Content-Type")).To(ContainSubstring("javascript"))
			})

			It("serves json with a json content type", func() {
				rec := request(http.MethodGet, "/docs/data.json")

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Header().Get("Content-Type")).To(ContainSubstring("json"))
			})

			It("answers HEAD with headers and no body", func() {
				rec := request(http.MethodHead, "/")

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Body.String()).To(BeEmpty())
			})
		})

		Describe("guarding the HTTP method", func() {
			It("rejects a write method with 405 and an Allow header", func() {
				rec := request(http.MethodPost, "/")

				Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed))
				Expect(rec.Header().Get("Allow")).To(Equal("GET, HEAD"))
			})
		})

		Describe("configuring the server", func() {
			It("falls back to a plain-text 404 when the not-found file is absent", func() {
				configured, err := eventcatalog.NewStaticServer(
					staticSiteFS(),
					eventcatalog.WithNotFoundFile("missing.html"),
				)
				Expect(err).ToNot(HaveOccurred())

				rec := httptest.NewRecorder()
				configured.Handler().
					ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

				Expect(rec.Code).To(Equal(http.StatusNotFound))
				Expect(rec.Body.String()).To(ContainSubstring("404 page not found"))
			})

			It("honours a custom immutable asset prefix", func() {
				configured, err := eventcatalog.NewStaticServer(
					staticSiteFS(),
					eventcatalog.WithImmutableAssetPrefixes("docs/"),
				)
				Expect(err).ToNot(HaveOccurred())

				rec := httptest.NewRecorder()
				configured.Handler().ServeHTTP(
					rec,
					httptest.NewRequest(http.MethodGet, "/docs/data.json", nil),
				)

				Expect(rec.Header().Get("Cache-Control")).To(ContainSubstring("immutable"))
			})
		})

		Describe("serving a tree baked into the binary", func() {
			It("serves a //go:embed tree through fs.Sub", func() {
				sub, err := fs.Sub(embeddedDist, "testdata/site")
				Expect(err).ToNot(HaveOccurred())

				configured, err := eventcatalog.NewStaticServer(sub)
				Expect(err).ToNot(HaveOccurred())

				rec := httptest.NewRecorder()
				configured.Handler().ServeHTTP(
					rec,
					httptest.NewRequest(http.MethodGet, "/docs/foo/", nil),
				)

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Body.String()).To(ContainSubstring(fooMarker))
			})
		})
	})
})
