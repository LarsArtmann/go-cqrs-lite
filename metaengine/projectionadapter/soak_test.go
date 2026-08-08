package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type soakItem struct {
	ID   string
	Name string
}

type soakQuery struct{ ID string }

func TestSoak_RecordPipeline_100K(t *testing.T) {
	if os.Getenv("SOAK_SKIP_RECORD") == "1" {
		t.Skip("SOAK_SKIP_RECORD=1")
	}

	if testing.Short() {
		t.Skip("soak test — not short")
	}

	const numEvents = 100_000
	const numKeys = 1_000

	q := metaengine.Query[soakQuery, soakItem](
		"soak-items",
		metaengine.On(soakItem{}, func(e soakItem) (string, soakItem) {
			return e.ID, e
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	decoder := func(_ string, payload []byte) (any, error) {
		var item soakItem
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		return item, nil
	}

	adapter := projectionadapter.New("soak", store, decoder)
	ctx := context.Background()

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	streamID := id.NewStreamID()

	for i := 0; i < numEvents; i++ {
		key := fmt.Sprintf("item-%d", i%numKeys)
		payload, _ := json.Marshal(soakItem{ID: key, Name: fmt.Sprintf("name-%d", i)})

		evt, err := event.NewEvent(
			event.Type("soakItem"), streamID, "Item",
			event.Version(i+1), payload,
		)
		if err != nil {
			t.Fatalf("event %d: %v", i, err)
		}

		if err := adapter.Handle(ctx, evt); err != nil {
			t.Fatalf("Handle %d: %v", i, err)
		}
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	totalAllocDelta := after.TotalAlloc - before.TotalAlloc
	heapInUseDelta := int64(after.HeapInuse) - int64(before.HeapInuse)

	t.Logf("100K events through AsRecord → Handle → ApplyRecord:")
	t.Logf("  TotalAlloc delta: %d bytes (%.1f MB)", totalAllocDelta, float64(totalAllocDelta)/1e6)
	t.Logf("  HeapInuse delta:  %d bytes (%.1f MB)", heapInUseDelta, float64(heapInUseDelta)/1e6)
	t.Logf("  Allocs/Event:     %.0f bytes", float64(totalAllocDelta)/float64(numEvents))

	result, err := store.Execute(soakQuery{ID: "item-0"})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}
	item, ok := result.(soakItem)
	if !ok {
		t.Fatalf("result is %T, want soakItem", result)
	}
	if item.ID != "item-0" {
		t.Fatalf("expected item-0, got %s", item.ID)
	}

	if heapInUseDelta > 100*1024*1024 {
		t.Errorf(
			"heap growth %.1f MB exceeds 100 MB limit — possible memory leak",
			float64(heapInUseDelta)/1e6,
		)
	}
}
