package sqliteengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// The raw readers must obey the same flush-before-read rule as MapGet and the
// pushdown scans: a buffered planned write is visible to them inside the tx.
func TestPlannedBatch_RawReadersObserveBufferedRows(t *testing.T) {
	t.Parallel()

	store, eng, _ := newBatchTestStore(t, -1)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	raw, ok := eng.(metaengine.RawValueReader)
	if !ok {
		t.Fatal("engine does not implement RawValueReader")
	}

	scanner, ok := eng.(metaengine.RawScanReader)
	if !ok {
		t.Fatal("engine does not implement RawScanReader")
	}

	err := store.InTransaction(ctx, func(ctx context.Context) error {
		if err := mb.MapSet(ctx, "batch_tracks", int64(9901), spikeMF1Row{
			ID: 9901, Domain: "raw.example.net", CreatedUnix: 1,
		}); err != nil {
			return err
		}

		rawVal, found, err := raw.GetRawValue(ctx, "batch_tracks", int64(9901))
		if err != nil {
			return err
		}

		if !found {
			t.Error("GetRawValue inside tx must observe the buffered planned row")
		} else if len(rawVal) == 0 {
			t.Error("GetRawValue returned an empty row")
		}

		res, err := scanner.ScanRawValues(ctx, "batch_tracks", []metaengine.FilterSpec{
			{Column: "domain", Op: metaengine.FilterEq, Value: "raw.example.net"},
		}, nil, nil, 10)
		if err != nil {
			return err
		}

		if len(res.Items) != 1 {
			t.Errorf("ScanRawValues inside tx: got %d rows, want 1", len(res.Items))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("InTransaction: %v", err)
	}

	v, found, err := mb.MapGet(ctx, "batch_tracks", int64(9901))
	if err != nil || !found {
		t.Fatalf("post-commit read: found=%v err=%v", found, err)
	}

	if v.(map[string]any)["domain"] != "raw.example.net" { //nolint:forcetypeassert
		t.Fatalf("post-commit value mismatch: %+v", v)
	}
}
