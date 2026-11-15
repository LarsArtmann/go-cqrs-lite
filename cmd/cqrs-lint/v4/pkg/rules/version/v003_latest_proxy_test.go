package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// The proxy-resolution paths below latestMinorFromProxy's guards are
// unreachable through the wrapper under `go test` (testing.Testing() is
// always true), so these tests exercise the extracted live body directly.

func TestLatestMinorFromProxy_TestBinaryGuard(t *testing.T) {
	t.Setenv("CQRS_LINT_NO_PROXY_FETCH", "1")

	minor, ok := latestMinorFromProxy(context.Background(), "example.com/module")
	if ok || minor != 0 {
		t.Fatalf("test binaries must never hit the proxy; got (%d, %v)", minor, ok)
	}
}

func TestLatestMinorFromProxyLive_BypassNeverReachesProxy(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "must not be consulted", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	t.Setenv("GOPROXY", server.URL)
	t.Setenv("GONOPROXY", "private.example/*")
	t.Setenv("GOPRIVATE", "")

	if _, ok := latestMinorFromProxyLive(context.Background(), "private.example/secret/module"); ok {
		t.Fatal("a bypassed module must not resolve")
	}
	if hits.Load() != 0 {
		t.Fatalf("bypassed module leaked %d proxy request(s)", hits.Load())
	}
}

func TestLatestMinorFromProxyLive_CacheAndFetch(t *testing.T) {
	const cached = "cached.example/m"
	const poisoned = "poisoned.example/m"
	const fetched = "fetched.example/m"
	t.Cleanup(func() {
		latestByModuleCache.Delete(cached)
		latestByModuleCache.Delete(poisoned)
		latestByModuleCache.Delete(fetched)
	})

	t.Setenv("GONOPROXY", "")
	t.Setenv("GOPRIVATE", "")

	latestByModuleCache.Store(cached, 7)
	minor, ok := latestMinorFromProxyLive(context.Background(), cached)
	if !ok || minor != 7 {
		t.Fatalf("cache hit must answer without a request; got (%d, %v)", minor, ok)
	}

	latestByModuleCache.Store(poisoned, "not-an-int")
	if _, ok := latestMinorFromProxyLive(context.Background(), poisoned); ok {
		t.Fatal("a corrupted cache entry must not answer")
	}

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Version":"v4.9.0"}`))

			return
		}
		http.Error(w, "cache must absorb the second call", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)
	t.Setenv("GOPROXY", server.URL)

	minor, ok = latestMinorFromProxyLive(context.Background(), fetched)
	if !ok || minor != 9 {
		t.Fatalf("successful fetch must answer v4.9.0 → 9; got (%d, %v)", minor, ok)
	}

	minor, ok = latestMinorFromProxyLive(context.Background(), fetched)
	if !ok || minor != 9 {
		t.Fatalf("second resolve must come from the cache; got (%d, %v)", minor, ok)
	}
	if hits.Load() != 1 {
		t.Fatalf("cache must absorb the second request; saw %d hits", hits.Load())
	}
}

func TestLatestMinorFromProxyLive_FetchFailureIsNotCached(t *testing.T) {
	const failing = "failing.example/m"
	t.Cleanup(func() { latestByModuleCache.Delete(failing) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	t.Setenv("GOPROXY", server.URL)
	t.Setenv("GONOPROXY", "")
	t.Setenv("GOPRIVATE", "")

	if _, ok := latestMinorFromProxyLive(context.Background(), failing); ok {
		t.Fatal("a 500 must not resolve")
	}
	if _, cached := latestByModuleCache.Load(failing); cached {
		t.Fatal("a failed fetch must not poison the cache")
	}
}

func TestFirstProxyBase(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want string
	}{
		{"unset defaults to proxy.golang.org", "", "https://proxy.golang.org"},
		{"direct alone carries no URL", "direct", "https://proxy.golang.org"},
		{"off alone carries no URL", "off", "https://proxy.golang.org"},
		{"first usable entry wins", "https://corp.proxy.example,https://other.example", "https://corp.proxy.example"},
		{"skips direct then takes the URL", "direct, https://second.example", "https://second.example"},
		{"empty entries are skipped", ",,https://third.example,", "https://third.example"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOPROXY", tc.env)

			if got := firstProxyBase(); got != tc.want {
				t.Errorf("firstProxyBase(GOPROXY=%q) = %q, want %q", tc.env, got, tc.want)
			}
		})
	}
}

func TestPathHasTrailingGlob(t *testing.T) {
	cases := []struct {
		pattern string
		want    bool
	}{
		{"example.com/private", false},
		{"example.com/*", true},
		{"example.com/pa?h", true},
		{"example.com/pa[h", true},
	}

	for _, tc := range cases {
		if got := pathHasTrailingGlob(tc.pattern); got != tc.want {
			t.Errorf("pathHasTrailingGlob(%q) = %v, want %v", tc.pattern, got, tc.want)
		}
	}
}

func TestGlobPrefixMatches(t *testing.T) {
	cases := []struct {
		name       string
		pattern    string
		modulePath string
		want       bool
	}{
		{"all elements match", "example.com/*", "example.com/foo", true},
		{"pattern is an element prefix", "example.com/foo", "example.com/foo/deeper/still", true},
		{"glob elements match in the middle", "example.com/*/sub", "example.com/foo/sub/x", true},
		{"pattern longer than path", "example.com/a/b/c", "example.com/a", false},
		{"element mismatch", "example.com/foo", "other.example/foo", false},
		{"glob element mismatch", "example.com/*.corp", "example.com/public", false},
		{"malformed glob element errors to no-match", "example.com/[", "example.com/x", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := globPrefixMatches(tc.pattern, tc.modulePath); got != tc.want {
				t.Errorf("globPrefixMatches(%q, %q) = %v, want %v", tc.pattern, tc.modulePath, got, tc.want)
			}
		})
	}
}

func TestModuleBypassesProxy_MalformedPatternIsNotABypass(t *testing.T) {
	t.Setenv("GONOPROXY", "example.com/[")
	t.Setenv("GOPRIVATE", "")

	if moduleBypassesProxy("example.com/x") {
		t.Fatal("a malformed pattern must fall through to no-bypass, not match everything")
	}
}

func TestFetchLatestMinorFromBase_ErrorBranches(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"non-200 status", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusGone)
		}},
		{"unparsable body", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`this is not json`))
		}},
		{"empty version", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"Version":""}`))
		}},
		{"truncated body", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "128")
			_, _ = w.Write([]byte(`{"Version":"v4`))
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			t.Cleanup(server.Close)

			if _, ok := fetchLatestMinorFromBase(context.Background(), server.URL, "example.com/m"); ok {
				t.Fatalf("%s must not resolve", strings.ToLower(tc.name))
			}
		})
	}

	t.Run("malformed base URL", func(t *testing.T) {
		if _, ok := fetchLatestMinorFromBase(context.Background(), "http://bad host", "example.com/m"); ok {
			t.Fatal("a malformed base URL must not resolve")
		}
	})
}

func TestFetchLatestMinorFromBase_SemaphoreCancellation(t *testing.T) {
	for i := 0; i < proxyFetchConcurrency; i++ {
		proxyFetchSlots <- struct{}{}
	}
	t.Cleanup(func() {
		for i := 0; i < proxyFetchConcurrency; i++ {
			<-proxyFetchSlots
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, ok := fetchLatestMinorFromBase(ctx, "https://proxy.example", "example.com/m"); ok {
		t.Fatal("a canceled context must not resolve while waiting for a slot")
	}
}
