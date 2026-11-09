package docserver

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// TestArchitectureCacheRenderOnce pins the cache contract directly: the
// same catalog instance renders exactly once, and a new instance re-renders.
func TestArchitectureCacheRenderOnce(t *testing.T) {
	cat := catalog.NewBuilder("Cache", "1.0.0").Build()

	renders := 0

	render := func(*catalog.Catalog) ([]byte, error) {
		renders++

		return []byte("rendered"), nil
	}

	var cache architectureCache

	for range 3 {
		got, err := cache.load(cat, render)
		if err != nil || string(got) != "rendered" {
			t.Fatalf("load: %q, %v", got, err)
		}
	}

	if renders != 1 {
		t.Errorf("the same instance rendered %d times, want 1", renders)
	}

	next := catalog.NewBuilder("Cache", "1.1.0").Build()

	if _, err := cache.load(next, render); err != nil {
		t.Fatalf("load after invalidation: %v", err)
	}

	if renders != 2 {
		t.Errorf("a new instance must re-render: %d renders, want 2", renders)
	}
}

// TestArchitectureCacheRendersErrorPins pins that a render error is cached
// with the instance, not retried per request.
func TestArchitectureCacheRendersErrorPins(t *testing.T) {
	cat := catalog.NewBuilder("Cache", "1.0.0").Build()

	renders := 0

	render := func(*catalog.Catalog) ([]byte, error) {
		renders++

		return nil, errors.New("boom")
	}

	var cache architectureCache

	for range 2 {
		if _, err := cache.load(cat, render); err == nil {
			t.Fatal("expected the cached error")
		}
	}

	if renders != 1 {
		t.Errorf("error renders must cache too: %d renders, want 1", renders)
	}
}
