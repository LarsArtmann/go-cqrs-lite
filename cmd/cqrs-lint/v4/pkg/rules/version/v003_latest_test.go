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

	if got := latestMinorFor("m", nil); got != latestKnownMinor {
		t.Errorf("nil fetch must fall back to the constant; got %d", got)
	}

	if got := latestMinorFor("m", func(context.Context, string) (int, bool) {
		return 0, false
	}); got != latestKnownMinor {
		t.Errorf("failed fetch must fall back to the constant; got %d", got)
	}

	if got := latestMinorFor("m", func(context.Context, string) (int, bool) {
		return 23, true
	}); got != 23 {
		t.Errorf("successful fetch must win; got %d", got)
	}
}
