package listing_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

var update = flag.Bool("update", false, "update golden files")

func testListingStatus(
	streamID id.StreamID,
	streamType string,
	v int,
	evtCount int,
	lastEventAt time.Time,
	status event.TombstoneStatus,
) listing.StreamStatus {
	return listing.StreamStatus{
		Ref: listing.StreamListing{
			ID:          streamID,
			Type:        id.StreamType(streamType),
			Version:     event.Version(v),
			EventCount:  uint(evtCount),
			LastEventAt: lastEventAt,
		},
		Status: status,
	}
}

func TestGolden_StreamStatusJSON(t *testing.T) {
	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	statuses := []listing.StreamStatus{
		testListingStatus(
			streamID,
			"User",
			10,
			10,
			time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			event.TombstoneActive,
		),
		testListingStatus(
			streamID,
			"Order",
			5,
			5,
			time.Date(2026, 5, 15, 8, 30, 0, 0, time.UTC),
			event.TombstoneTombstoned,
		),
		{
			Ref: listing.StreamListing{
				ID:          streamID,
				Type:        "Cart",
				Version:     event.Version(0),
				EventCount:  0,
				LastEventAt: time.Time{},
			},
			Status: event.TombstoneUndetermined,
		},
	}

	got, err := json.Marshal(statuses, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "stream-status.json"),
		got,
		*update,
	)
}

func TestGolden_PageJSON(t *testing.T) {
	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	page := listing.Page[listing.StreamStatus]{
		Items: []listing.StreamStatus{
			testListingStatus(
				streamID,
				"User",
				3,
				3,
				time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				event.TombstoneActive,
			),
		},
		HasMore: true,
	}

	got, err := json.Marshal(page, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(t, filepath.Join("testdata", "golden", "page.json"), got, *update)
}
