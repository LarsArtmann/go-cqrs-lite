package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEscapeModulePath(t *testing.T) {
	t.Parallel()

	got := escapeModulePath("github.com/larsartmann/go-cqrs-lite/event/v4")
	if got != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Errorf("lowercase path must pass through unchanged; got %q", got)
	}

	got = escapeModulePath("github.com/LarsArtmann/Module/v4")
	if got != "github.com/!lars!artmann/!module/v4" {
		t.Errorf("uppercase runes must escape as !lower; got %q", got)
	}
}

func TestFetchLatestMinorFromBase(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/github.com/larsartmann/go-cqrs-lite/systemcatalog/v4/@latest" {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Version":"v4.0.0","Time":"2026-07-01T00:00:00Z"}`))
	}))
	t.Cleanup(server.Close)

	minor, ok := fetchLatestMinorFromBase(context.Background(), server.URL,
		"github.com/larsartmann/go-cqrs-lite/systemcatalog/v4")
	if !ok || minor != 0 {
		t.Fatalf("systemcatalog @latest v4.0.0 → (0, true); got (%d, %v)", minor, ok)
	}
}

func TestFetchLatestMinorFromBase_NonV4AndGarbage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Version":"v5.1.0"}`))
	}))
	t.Cleanup(server.Close)

	if _, ok := fetchLatestMinorFromBase(context.Background(), server.URL,
		"github.com/larsartmann/go-cqrs-lite/event/v4"); ok {
		t.Fatal("a v5 latest must not answer a v4 minor")
	}

	if _, ok := fetchLatestMinorFromBase(context.Background(), server.URL+"/unreachable",
		"github.com/larsartmann/go-cqrs-lite/event/v4"); ok {
		t.Fatal("an unreachable base must not answer")
	}
}

func TestLatestMinorFor_Fallbacks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if got := latestMinorFor(ctx, "m", nil); got != latestKnownMinor {
		t.Errorf("nil fetch must fall back to the constant; got %d", got)
	}

	if got := latestMinorFor(ctx, "m", func(context.Context, string) (int, bool) {
		return 0, false
	}); got != latestKnownMinor {
		t.Errorf("failed fetch must fall back to the constant; got %d", got)
	}

	if got := latestMinorFor(ctx, "m", func(context.Context, string) (int, bool) {
		return 23, true
	}); got != 23 {
		t.Errorf("successful fetch must win; got %d", got)
	}
}

// TestModuleBypassesProxy pins the GOPRIVATE/GONOPROXY honor contract
// (M37/F232): private module paths never reach the public proxy — the
// request could not succeed and would leak the module name.
func TestModuleBypassesProxy(t *testing.T) {
	t.Setenv("GONOPROXY", "")
	t.Setenv("GOPRIVATE", "")

	cases := []struct {
		name       string
		gonoproxy  string
		goprivate  string
		module     string
		wantBypass bool
	}{
		{"no env set", "", "", "github.com/larsartmann/go-cqrs-lite/event/v4", false},
		{"GOPRIVATE exact", "", "example.com/private", "example.com/private", true},
		{"GOPRIVATE element prefix", "", "example.com/private", "example.com/private/more", true},
		{"GOPRIVATE no partial element", "", "example.com/private", "example.com/privateX", false},
		{"GOPRIVATE wildcard", "", "example.com/*.corp", "example.com/foo.corp", true},
		{"GOPRIVATE wildcard prefix", "", "example.com/*.corp", "example.com/foo.corp/sub", true},
		{"GOPRIVATE wildcard no match", "", "example.com/*.corp", "example.com/public", false},
		{"GONOPROXY overrides GOPRIVATE", "example.com/only", "example.com/other", "example.com/other", false},
		{"GONOPROXY matches", "example.com/only", "example.com/other", "example.com/only", true},
		{"comma-separated list", "a.com/x, b.com/y", "", "b.com/y/deep", true},
		{"different host", "", "corp.example/internal", "github.com/other/module", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GONOPROXY", tc.gonoproxy)
			t.Setenv("GOPRIVATE", tc.goprivate)

			if got := moduleBypassesProxy(tc.module); got != tc.wantBypass {
				t.Errorf("moduleBypassesProxy(%q) = %v, want %v", tc.module, got, tc.wantBypass)
			}
		})
	}
}
