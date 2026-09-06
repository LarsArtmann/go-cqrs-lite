package docserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCSPBrowser_NoViolations validates the opt-in CSP policy with a REAL
// headless Chromium against the embedded Scalar and AsyncAPI bundles: every
// HTML page must render with zero CSP refusals on the browser's console, and
// the self-hosted bundle assets must be fetched successfully.
//
// Skipped unless CQRS_BROWSER names a chromium binary — `nix run .#check-csp`
// wires it up; CI can do the same.
func TestCSPBrowser_NoViolations(t *testing.T) {
	browser := os.Getenv("CQRS_BROWSER")
	if browser == "" {
		t.Skip("set CQRS_BROWSER to a chromium binary to run the browser CSP validation")
	}

	ds := NewDocsServer(testProvider, Config{
		ServiceName: "Test Service",
		Version:     "1.0.0",
		Description: "A test service",
		EnableCSP:   true,
	})

	mux := http.NewServeMux()
	ds.RegisterRoutes(mux)

	fetches := newFetchLog()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		mux.ServeHTTP(rec, r)
		fetches.record(r.URL.Path, rec.status)
	}))
	t.Cleanup(srv.Close)

	pages := []struct {
		name         string
		path         string
		mustFetch    []string
		domReference string
	}{
		{
			name:         "index",
			path:         "/docs",
			domReference: "Test Service",
		},
		{
			name:         "openapi-ui",
			path:         "/docs/openapi",
			mustFetch:    []string{"/docs/static/scalar.js"},
			domReference: "openapi",
		},
		{
			name:         "asyncapi-ui",
			path:         "/docs/asyncapi",
			mustFetch:    []string{"/docs/static/asyncapi-react.js"},
			domReference: "asyncapi",
		},
		{
			name:         "d2-view",
			path:         "/docs/d2",
			domReference: "Architecture Diagram",
		},
	}

	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			dom, console := renderWithBrowser(t, browser, srv.URL+page.path)

			for _, refusal := range []string{"Refused to", "violates Content Security Policy"} {
				if strings.Contains(console, refusal) {
					t.Errorf("browser reported a CSP refusal:\n%s", console)
				}
			}

			if !strings.Contains(dom, page.domReference) {
				t.Errorf("rendered DOM does not reference %q:\n%.500s", page.domReference, dom)
			}

			for _, asset := range page.mustFetch {
				if status := fetches.status(asset); status != http.StatusOK {
					t.Errorf("browser fetched %s with status %d", asset, status)
				}
			}
		})
	}
}

// renderWithBrowser runs headless Chromium against the URL and returns the
// rendered DOM (stdout) plus the console log (stderr, where CSP refusals
// surface).
func renderWithBrowser(t *testing.T, browser, url string) (string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, browser,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--enable-logging=stderr",
		"--v=0",
		"--virtual-time-budget=10000",
		"--dump-dom",
		url,
	)

	var stdout, stderr strings.Builder

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("headless browser run: %v\nstderr:\n%s", err, stderr.String())
	}

	return stdout.String(), stderr.String()
}

// statusRecorder captures the response status so the test can assert the
// browser actually fetched the embedded bundle assets.
type statusRecorder struct {
	http.ResponseWriter

	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
}

// fetchLog records URL path → last response status, safe for the concurrent
// requests a real browser issues.
type fetchLog struct {
	mu sync.Mutex
	m  map[string]int
}

func newFetchLog() *fetchLog {
	return &fetchLog{m: make(map[string]int)}
}

func (f *fetchLog) record(path string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.m[path] = status
}

func (f *fetchLog) status(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.m[path]
}
