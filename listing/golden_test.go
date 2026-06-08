package listing_test

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_AggregateStatusJSON(t *testing.T) {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	statuses := []listing.AggregateStatus{
		{
			Ref: listing.AggregateListing{
				ID:          aggID,
				Type:        "User",
				Version:     event.Version(10),
				EventCount:  10,
				LastEventAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			},
			Status: event.TombstoneActive,
		},
		{
			Ref: listing.AggregateListing{
				ID:          aggID,
				Type:        "Order",
				Version:     event.Version(5),
				EventCount:  5,
				LastEventAt: time.Date(2026, 5, 15, 8, 30, 0, 0, time.UTC),
			},
			Status: event.TombstoneTombstoned,
		},
		{
			Ref: listing.AggregateListing{
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

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "aggregate-status.json"),
		got,
		*update,
	)
}

func TestGolden_PageJSON(t *testing.T) {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	page := listing.Page[listing.AggregateStatus]{
		Items: []listing.AggregateStatus{
			{
				Ref: listing.AggregateListing{
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

	got, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(t, filepath.Join("testdata", "golden", "page.json"), got, *update)
}
