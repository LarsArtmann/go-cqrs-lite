package docserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDocsServer_RoutesAreETagged304 pins the conditional-request wrap on
// RegisterRoutes: the deterministic spec/data routes carry a content-hash
// ETag, and a revalidation with If-None-Match answers 304 with no body —
// revisits revalidate instead of re-downloading the payload. The
// templ-rendered HTML pages are deliberately unwrapped: their per-request
// CSP nonce legitimately changes the bytes every render.
func TestDocsServer_RoutesAreETagged304(t *testing.T) {
	srv := testServer(t)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	paths := []string{
		"/docs/openapi.json",
		"/docs/openapi.yaml",
		"/docs/asyncapi.json",
		"/docs/asyncapi.yaml",
		"/docs/catalog.json",
		"/docs/d2.txt",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + path)
			if err != nil {
				t.Fatalf("first GET: %v", err)
			}
			_ = resp.Body.Close()

			tag := resp.Header.Get("ETag")
			if tag == "" {
				t.Fatalf("first GET: no ETag header (status %d)", resp.StatusCode)
			}

			req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("If-None-Match", tag)

			revalidate, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("revalidation GET: %v", err)
			}
			defer func() { _ = revalidate.Body.Close() }()

			if revalidate.StatusCode != http.StatusNotModified {
				t.Fatalf("revalidation: status %d, want %d", revalidate.StatusCode, http.StatusNotModified)
			}

			body, err := io.ReadAll(revalidate.Body)
			if err != nil {
				t.Fatalf("read revalidation body: %v", err)
			}

			if len(body) != 0 {
				t.Fatalf("revalidation: 304 carried %d body bytes", len(body))
			}
		})
	}
}
