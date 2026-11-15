package metaengine_test

// Store-level planned-table sweep dispatch tests (dnsblockd metaflood
// MF3.2): unknown queries are refused with ErrNoQueryForName, and engines
// without the PlannedTableSweeper capability (memory) are refused loudly
// instead of silently trimming nothing.

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type trimProbeRow struct {
	CreatedUnix int64 `json:"createdUnix"`
}

type trimProbeInput struct {
	CreatedUnix int64 `json:"createdUnix"`
}

func newTrimProbeStore(t *testing.T) *metaengine.Store {
	t.Helper()

	q := metaengine.Query[trimProbeInput, trimProbeRow]("trim_probe",
		metaengine.FilterOnField[trimProbeRow]("createdUnix", metaengine.FilterGe),
		metaengine.OnRecordTyped("trim.probe", trimProbeRow{},
			func(_ record.Record, e trimProbeRow) (int64, trimProbeRow) { return e.CreatedUnix, e }),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func TestTrimPlannedBefore_UnknownQueryRefused(t *testing.T) {
	t.Parallel()

	store := newTrimProbeStore(t)

	_, err := store.TrimPlannedBefore(context.Background(), "no_such_query", "createdUnix", 1)
	if !errors.Is(err, metaengine.ErrNoQueryForName) {
		t.Fatalf("expected ErrNoQueryForName, got %v", err)
	}
}

// TestTrimPlannedBefore_CapabilityRefusedLoudly: an engine WITHOUT the
// PlannedTableSweeper capability (the memory engine) must be refused with an
// explicit error — never a silent zero-row sweep that looks like success.
func TestTrimPlannedBefore_CapabilityRefusedLoudly(t *testing.T) {
	t.Parallel()

	store := newTrimProbeStore(t)

	if _, err := store.TrimPlannedBefore(context.Background(), "trim_probe", "createdUnix", 1); err == nil {
		t.Fatal("memory engine has no PlannedTableSweeper capability: the sweep must error, not no-op")
	}
}
