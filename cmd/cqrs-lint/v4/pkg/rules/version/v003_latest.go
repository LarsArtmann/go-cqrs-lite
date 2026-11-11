package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// fetchLatestMinorFromBase performs one GET <base>/<module>/@latest and
// parses the minor out of the returned Version.
func fetchLatestMinorFromBase(ctx context.Context, base, modulePath string) (int, bool) {
	ctx, cancel := context.WithTimeout(ctx, proxyFetchTimeout)
	defer cancel()

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
