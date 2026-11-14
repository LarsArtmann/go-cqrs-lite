package version

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"
)

// Per-module "latest" resolution (issue #129). go-cqrs-lite tags are
// PER-MODULE: metaengine reached v4.23.0 while systemcatalog stopped at
// v4.0.0, so a fleet-wide latestKnownMinor constant mislabels any module
// whose own version line never reached it — the suggested
// `go get ...@v4.3.0` does not even resolve. The proxy is asked for each
// module path's real @latest; every failure (offline, proxy error, test
// binary) falls back to the constant, which keeps today's behavior as the
// floor.

// proxyFetchTimeout bounds one @latest request; the rule must never turn a
// lint run into a network wait.
const proxyFetchTimeout = 1500 * time.Millisecond

// proxyFetchConcurrency caps how many @latest requests may be in flight at
// once: a fleet-scale monorepo lint consults one module path per rule, and
// unbounded parallelism would turn the proxy into a burst.
const proxyFetchConcurrency = 4

// proxyFetchSlots is the in-flight cap; a struct{} channel is a semaphore.
//
//nolint:gochecknoglobals // process-lifetime semaphore, fixed capacity
var proxyFetchSlots = make(chan struct{}, proxyFetchConcurrency)

// latestByModuleCache memoizes proxy answers for the process lifetime —
// one module path is asked at most once per run however many rules consult
// it.
//
//nolint:gochecknoglobals // process-lifetime read-through cache, set once per module path
var latestByModuleCache sync.Map // string → int

// latestMinorFetch is the seam detector tests stub; the default respects
// the test-binary and kill-switch guards below.
//
//nolint:gochecknoglobals // injectable seam, set once per test
var latestMinorFetch = latestMinorFromProxy

// latestMinorFor returns the module's latest v4 minor: the proxy's answer
// when one is available, otherwise the fleet-wide fallback constant.
func latestMinorFor(
	ctx context.Context,
	modulePath string,
	fetch func(context.Context, string) (int, bool),
) int {
	if fetch == nil {
		return latestKnownMinor
	}

	if minor, ok := fetch(ctx, modulePath); ok && minor >= 0 {
		return minor
	}

	return latestKnownMinor
}

// latestMinorFromProxy resolves a module's latest minor from the Go module
// proxy. Guards: test binaries never hit the network (hermetic tests), and
// CQRS_LINT_NO_PROXY_FETCH=1 is the consumer escape hatch for sandboxed
// environments. Unreachable proxies answer (0, false) — the caller falls
// back.
func latestMinorFromProxy(ctx context.Context, modulePath string) (int, bool) {
	if testing.Testing() || os.Getenv("CQRS_LINT_NO_PROXY_FETCH") == "1" {
		return 0, false
	}

	if moduleBypassesProxy(modulePath) {
		return 0, false
	}

	if cached, ok := latestByModuleCache.Load(modulePath); ok {
		if minor, ok := cached.(int); ok {
			return minor, true
		}

		return 0, false
	}

	base := firstProxyBase()
	if base == "" {
		return 0, false
	}

	minor, ok := fetchLatestMinorFromBase(ctx, base, modulePath)
	if !ok {
		return 0, false
	}

	latestByModuleCache.Store(modulePath, minor)

	return minor, true
}

// moduleBypassesProxy reports whether a module path matches GONOPROXY (or,
// when that is unset, GOPRIVATE — Go's documented default chain). Private
// modules must never be sent to a public proxy: the request cannot succeed
// (404 at best) and leaks the module name to the proxy operator. Bypassed
// modules use the fleet fallback constant.
func moduleBypassesProxy(modulePath string) bool {
	patterns := strings.FieldsFunc(os.Getenv("GONOPROXY"), func(r rune) bool {
		return r == ',' || r == ' '
	})
	if len(patterns) == 0 {
		patterns = strings.FieldsFunc(os.Getenv("GOPRIVATE"), func(r rune) bool {
			return r == ',' || r == ' '
		})
	}

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		// The go command's module-path matching: a pattern matches when the
		// module path equals the pattern or has it as a path-element prefix
		// (element boundaries are "/"), with path.Match glob semantics inside
		// each element.
		if path.Match(pattern, modulePath) {
			return true
		}

		if strings.HasPrefix(modulePath, pattern+"/") ||
			(pathHasTrailingGlob(pattern) && globPrefixMatches(pattern, modulePath)) {
			return true
		}
	}

	return false
}

// pathHasTrailingGlob reports whether the pattern's last element contains a
// glob metacharacter, in which case the prefix rule needs element-wise
// matching instead of a plain string prefix.
func pathHasTrailingGlob(pattern string) bool {
	return strings.ContainsAny(pattern, `*?[`)
}

// globPrefixMatches element-wise matches a glob pattern against a module
// path: every pattern element must match the corresponding path element,
// and the pattern must not be LONGER than the path.
func globPrefixMatches(pattern, modulePath string) bool {
	patElems := strings.Split(pattern, "/")
	pathElems := strings.Split(modulePath, "/")
	if len(patElems) > len(pathElems) {
		return false
	}

	for i, pe := range patElems {
		match, err := path.Match(pe, pathElems[i])
		if err != nil || !match {
			return false
		}
	}

	return true
}

// fetchLatestMinorFromBase performs one GET <base>/<module>/@latest and
// parses the minor out of the returned Version.
func fetchLatestMinorFromBase(ctx context.Context, base, modulePath string) (int, bool) {
	ctx, cancel := context.WithTimeout(ctx, proxyFetchTimeout)
	defer cancel()

	// Cap concurrent proxy requests across all in-flight rules.
	select {
	case proxyFetchSlots <- struct{}{}:
		defer func() { <-proxyFetchSlots }()
	case <-ctx.Done():
		return 0, false
	}

	url := strings.TrimSuffix(base, "/") + "/" + escapeModulePath(modulePath) + "/@latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return 0, false
	}

	var latest struct {
		Version string `json:"Version"` //nolint:tagliatelle // the module proxy wire contract spells the field "Version"
	}
	if err := json.Unmarshal(body, &latest); err != nil || latest.Version == "" {
		return 0, false
	}

	major, minor, ok := majorMinorVersion(latest.Version)
	if !ok || major != 4 {
		return 0, false
	}

	return minor, true
}

// firstProxyBase returns the first usable GOPROXY entry: "direct" and
// "off" carry no proxy URL and are skipped. The documented default is
// proxy.golang.org.
func firstProxyBase() string {
	for _, entry := range strings.FieldsFunc(os.Getenv("GOPROXY"), func(r rune) bool {
		return r == ',' || r == ' '
	}) {
		if entry == "direct" || entry == "off" || entry == "" {
			continue
		}

		return entry
	}

	return "https://proxy.golang.org"
}

// escapeModulePath applies the module proxy's path escaping: every
// uppercase rune becomes "!"+lowercase (go-cqrs-lite paths are lowercase,
// but forks need not be).
func escapeModulePath(path string) string {
	var b strings.Builder

	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))

			continue
		}

		b.WriteRune(r)
	}

	return b.String()
}

// renderLatestMinor formats the minor for the finding message ("v4.23.x").
func renderLatestMinor(minor int) string { return fmt.Sprintf("v4.%d.x", minor) }
