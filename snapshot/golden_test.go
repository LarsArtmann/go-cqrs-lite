package snapshot_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_SnapshotStructure(t *testing.T) {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	state, err := json.Marshal(map[string]string{
		"name":  "Alice",
		"email": "alice@example.com",
		"items": "widget,gadget",
	})
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}

	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       event.Version(5),
		State:         state,
		CreatedAt:     time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	got, err := json.MarshalIndent(struct {
		AggregateID   string `json:"aggregateId"`
		AggregateType string `json:"aggregateType"`
		Version       int    `json:"version"`
		State         string `json:"state"`
		CreatedAt     string `json:"createdAt"`
	}{
		AggregateID:   snap.AggregateID.String(),
		AggregateType: string(snap.AggregateType),
		Version:       snap.Version.Int(),
		State:         string(snap.State),
		CreatedAt:     snap.CreatedAt.Format(time.RFC3339Nano),
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertSnapshotGolden(t, filepath.Join("testdata", "golden", "snapshot-structure.json"), got)
}

func TestGolden_EveryNEventsStrategy(t *testing.T) {
	strategy := snapshot.MustEveryNEvents(3)

	type entry struct {
		Version int  `json:"version"`
		Snap    bool `json:"shouldSnapshot"`
	}

	entries := make([]entry, 21)
	for i := 0; i <= 20; i++ {
		entries[i] = entry{
			Version: i,
			Snap:    strategy.ShouldSnapshot("User", event.Version(i)),
		}
	}

	got, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertSnapshotGolden(
		t,
		filepath.Join("testdata", "golden", "every-n-events-strategy.json"),
		got,
	)
}

func assertSnapshotGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", path, err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch for %s (run with -update to refresh)", path)
	}
}
