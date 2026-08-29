package docserver

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// cspNonceFromHeader extracts the nonce from the script-src 'nonce-...'
// source expression of a Content-Security-Policy header.
func cspNonceFromHeader(t *testing.T, header string) string {
	t.Helper()

	re := regexp.MustCompile(`'nonce-([^']+)'`)
	m := re.FindStringSubmatch(header)
	if m == nil {
		t.Fatalf("no nonce source in CSP header: %q", header)
	}

	return m[1]
}

func TestDocsServer_CSP_OptIn(t *testing.T) {
	t.Parallel()

	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
		EnableCSP:   true,
	})
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for _, path := range []string{"/docs", "/docs/openapi", "/docs/asyncapi", "/docs/d2"} {
		req := newTestRequest(path)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d", path, recorder.Code)
		}

		policy := recorder.Header().Get("Content-Security-Policy")
		if policy == "" {
			t.Fatalf("GET %s: expected Content-Security-Policy header", path)
		}

		nonce := cspNonceFromHeader(t, policy)

		if !strings.Contains(policy, "script-src 'self' 'nonce-"+nonce+"'") {
			t.Errorf("GET %s: policy must gate scripts to self + nonce, got %q", path, policy)
		}

		body := recorder.Body.String()
		if !strings.Contains(body, `nonce="`+nonce+`"`) {
			t.Errorf("GET %s: expected script tags stamped with the header nonce %q", path, nonce)
		}
	}
}

func TestDocsServer_CSP_DisabledByDefault(t *testing.T) {
	t.Parallel()

	srv := testServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := newTestRequest("/docs/openapi")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("expected no CSP header by default, got %q", got)
	}

	if !strings.Contains(recorder.Body.String(), `nonce="`) {
		t.Error("expected inert nonce attributes on script tags even without the header")
	}
}

func TestDocsServer_CSP_NonceUniquePerRequest(t *testing.T) {
	t.Parallel()

	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
		EnableCSP:   true,
	})
	handler := srv.OpenAPIUI()

	nonces := make([]string, 0, 2)
	for range 2 {
		recorder := httptest.NewRecorder()
		handler(recorder, newTestRequest("/docs/openapi"))

		nonces = append(
			nonces,
			cspNonceFromHeader(t, recorder.Header().Get("Content-Security-Policy")),
		)
	}

	if nonces[0] == nonces[1] {
		t.Errorf("expected a fresh nonce per request, both were %q", nonces[0])
	}
}
