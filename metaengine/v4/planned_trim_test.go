package metaengine_test

// Store-level planned-table sweep dispatch tests (dnsblockd metaflood
// MF3.2): unknown queries are refused with ErrNoQueryForName, and engines
// without the PlannedTableSweeper capability (memory) are refused loudly
// instead of silently trimming nothing.

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/memoryengine"
)

func TestTrimPlannedBefore_UnknownQueryRefused(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan([]metaengine.Engine{memoryengine.New()})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	_, err = store.TrimPlannedBefore(context.Background(), "no_such_query", "createdUnix", 1)
	if !errors.Is(err, metaengine.ErrNoQueryForName) {
		t.Fatalf("expected ErrNoQueryForName, got %v", err)
	}
}
