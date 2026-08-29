package docserver

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/a-h/templ"
)

// cspHeaderKey is the response header carrying the Content-Security-Policy.
const cspHeaderKey = "Content-Security-Policy"

// cspNonceBytes is the CSP nonce entropy: 128 bits, the CSP Spec-recommended
// minimum for unpredictability.
const cspNonceBytes = 16

// newNonce mints a per-request CSP nonce: 128 bits of CSPRNG entropy,
// base64-encoded without padding so the value is valid both as a script
// nonce attribute and inside the 'nonce-...' CSP source expression.
// The bool is false only if the system CSPRNG fails; callers then render
// without a nonce (and must not send a nonce-gated CSP header).
func newNonce() (string, bool) {
	b := make([]byte, cspNonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}

	return base64.RawStdEncoding.EncodeToString(b), true
}

// cspHeaderValue builds the policy for docserver pages. Scripts are
// self-hosted bundles or nonce-gated inline bootstrap code; styles allow
// inline because the embedded Scalar and AsyncAPI bundles inject styles at
// runtime and cannot carry a nonce. Everything else is locked to same-origin.
func cspHeaderValue(nonce string) string {
	return "default-src 'self'; " +
		"script-src 'self' 'nonce-" + nonce + "'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"font-src 'self' data:; " +
		"connect-src 'self'; " +
		"base-uri 'none'; " +
		"frame-ancestors 'none'; " +
		"form-action 'none'; " +
		"object-src 'none'"
}

// applyCSP attaches a fresh per-request nonce to the request context (templ
// components read it back with templ.GetNonce to stamp script attributes)
// and, when EnableCSP is set, sends the matching policy header. Without a
// usable nonce the request passes through untouched — the pages render
// exactly as they did before CSP support.
func (ds *DocsServer) applyCSP(w http.ResponseWriter, r *http.Request) *http.Request {
	nonce, ok := newNonce()
	if !ok {
		return r
	}

	if ds.config.EnableCSP {
		w.Header().Set(cspHeaderKey, cspHeaderValue(nonce))
	}

	return r.WithContext(templ.WithNonce(r.Context(), nonce))
}
