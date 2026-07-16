package snapshot_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func everyN(tb testing.TB, n int) snapshot.SnapshotStrategy {
	tb.Helper()

	s, err := snapshot.EveryNEvents(n)
	if err != nil {
		tb.Fatalf("snapshot every-n %d: %v", n, err)
	}

	return s
}

var update = flag.Bool("update", false, "update golden files")

func TestGolden_SnapshotStructure(t *testing.T) {
	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	state, err := json.Marshal(struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Items string `json:"items"`
	}{
		Name:  "Alice",
		Email: "alice@example.com",
		Items: "widget,gadget",
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

	got, err := json.Marshal(struct {
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
	}, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "snapshot-structure.json"),
		got,
		*update,
	)
}

func TestGolden_EveryNEventsStrategy(t *testing.T) {
	strategy := everyN(t, 3)

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

	got, err := json.Marshal(entries, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "every-n-events-strategy.json"),
		got,
		*update,
	)
}
