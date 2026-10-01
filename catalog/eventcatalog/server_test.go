package eventcatalog_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

func testStaticFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":          {Data: []byte("home")},
		"404.html":            {Data: []byte("custom not found")},
		"legacy.html":         {Data: []byte("legacy")},
		"docs/foo/index.html": {Data: []byte("foo")},
		"docs/data.json":      {Data: []byte(`{"ok":true}`)},
		"_astro/app.js":       {Data: []byte("console.log(1)")},
	}
}

func newStaticServer(t *testing.T, opts ...eventcatalog.StaticOption) *eventcatalog.StaticServer {
	t.Helper()

	srv, err := eventcatalog.NewStaticServer(testStaticFS(), opts...)
	if err != nil {
		t.Fatalf("NewStaticServer: %v", err)
	}

	return srv
}

func staticRequest(
	t *testing.T,
	srv *eventcatalog.StaticServer,
	method, target string,
) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(method, target, nil))

	return rec
}

func TestStaticServerRouting(t *testing.T) {
	t.Parallel()

	srv := newStaticServer(t)

	tests := []struct {
		name         string
		method       string
		target       string
		wantStatus   int
		wantContains string
		wantLocation string
	}{
		{"root index", http.MethodGet, "/", http.StatusOK, "home", ""},
		{"explicit index", http.MethodGet, "/index.html", http.StatusOK, "home", ""},
		{"directory index", http.MethodGet, "/docs/foo/", http.StatusOK, "foo", ""},
		{
			"trailing slash redirect",
			http.MethodGet,
			"/docs/foo",
			http.StatusMovedPermanently,
			"",
			"/docs/foo/",
		},
		{"extensionless html", http.MethodGet, "/legacy", http.StatusOK, "legacy", ""},
		{"explicit html", http.MethodGet, "/legacy.html", http.StatusOK, "legacy", ""},
		{"custom 404", http.MethodGet, "/missing", http.StatusNotFound, "custom not found", ""},
		{
			"directory without index",
			http.MethodGet,
			"/docs",
			http.StatusNotFound,
			"custom not found",
			"",
		},
		{
			"traversal rejected",
			http.MethodGet,
			"/docs/../404.html",
			http.StatusNotFound,
			"custom not found",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := staticRequest(t, srv, tt.method, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantContains != "" && !strings.Contains(rec.Body.String(), tt.wantContains) {
				t.Fatalf("body = %q, want to contain %q", rec.Body.String(), tt.wantContains)
			}
			if loc := rec.Header().Get("Location"); loc != tt.wantLocation {
				t.Fatalf("Location = %q, want %q", loc, tt.wantLocation)
			}
		})
	}
}

func TestStaticServerHeaders(t *testing.T) {
	t.Parallel()

	srv := newStaticServer(t)

	t.Run("html is not cached", func(t *testing.T) {
		t.Parallel()

		rec := staticRequest(t, srv, http.MethodGet, "/")
		if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("Cache-Control = %q, want no-cache", got)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("Content-Type = %q, want html", ct)
		}
	})

	t.Run("astro bundle is immutable", func(t *testing.T) {
		t.Parallel()

		rec := staticRequest(t, srv, http.MethodGet, "/_astro/app.js")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
			t.Fatalf("Cache-Control = %q, want immutable", got)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
			t.Fatalf("Content-Type = %q, want javascript", ct)
		}
	})

	t.Run("head has no body", func(t *testing.T) {
		t.Parallel()

		rec := staticRequest(t, srv, http.MethodHead, "/")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("HEAD body = %q, want empty", rec.Body.String())
		}
	})
}

func TestStaticServerRejectsNonGET(t *testing.T) {
	t.Parallel()

	srv := newStaticServer(t)

	rec := staticRequest(t, srv, http.MethodPost, "/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", allow)
	}
}

func TestStaticServerOptions(t *testing.T) {
	t.Parallel()

	t.Run("custom not-found file", func(t *testing.T) {
		t.Parallel()

		srv := newStaticServer(t, eventcatalog.WithNotFoundFile("missing-404.html"))
		if rec := staticRequest(t, srv, http.MethodGet, "/nope"); rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("custom immutable prefix", func(t *testing.T) {
		t.Parallel()

		srv := newStaticServer(t, eventcatalog.WithImmutableAssetPrefixes("docs/"))
		rec := staticRequest(t, srv, http.MethodGet, "/docs/data.json")
		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
			t.Fatalf("Cache-Control = %q, want immutable", got)
		}
	})
}

func TestNewStaticServerRejectsNilFS(t *testing.T) {
	t.Parallel()

	if _, err := eventcatalog.NewStaticServer(nil); err == nil {
		t.Fatal("NewStaticServer(nil) = nil error, want error")
	}
}
