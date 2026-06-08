package listing_test

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
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_AggregateStatusJSON(t *testing.T) {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	statuses := []listing.AggregateStatus{
		{
			Ref: listing.AggregateRef{
				ID:          aggID,
				Type:        "User",
				Version:     event.Version(10),
				EventCount:  10,
				LastEventAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			},
			Status: event.TombstoneActive,
		},
		{
			Ref: listing.AggregateRef{
				ID:          aggID,
				Type:        "Order",
				Version:     event.Version(5),
				EventCount:  5,
				LastEventAt: time.Date(2026, 5, 15, 8, 30, 0, 0, time.UTC),
			},
			Status: event.TombstoneTombstoned,
		},
		{
			Ref: listing.AggregateRef{
				ID:          aggID,
				Type:        "Cart",
				Version:     event.Version(0),
				EventCount:  0,
				LastEventAt: time.Time{},
			},
			Status: event.TombstoneUndetermined,
		},
	}

	got, err := json.MarshalIndent(statuses, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertListingGolden(t, filepath.Join("testdata", "golden", "aggregate-status.json"), got)
}

func TestGolden_PageJSON(t *testing.T) {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	page := listing.Page[listing.AggregateStatus]{
		Items: []listing.AggregateStatus{
			{
				Ref: listing.AggregateRef{
					ID:          aggID,
					Type:        "User",
					Version:     event.Version(3),
					EventCount:  3,
					LastEventAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				},
				Status: event.TombstoneActive,
			},
		},
		HasMore: true,
	}

	serializable := struct {
		Items   []json.RawMessage `json:"items"`
		HasMore bool              `json:"hasMore"`
	}{
		HasMore: page.HasMore,
	}

	for _, item := range page.Items {
		raw, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("marshal item: %v", err)
		}

		serializable.Items = append(serializable.Items, raw)
	}

	got, err := json.MarshalIndent(serializable, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	assertListingGolden(t, filepath.Join("testdata", "golden", "page.json"), got)
}

func assertListingGolden(t *testing.T, path string, got []byte) {
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
