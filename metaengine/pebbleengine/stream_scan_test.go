package pebbleengine

import (
	"context"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPebbleStreamScan_Unsorted verifies lazy iteration without sort.
func TestPebbleStreamScan_Unsorted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for _, item := range []struct {
		key   string
		score int
	}{
		{"a", 10}, {"b", 20}, {"c", 5}, {"d", 30},
	} {
		_ = mb.MapSet(ctx, "items", item.key, map[string]any{
			"score": item.score,
			"name":  item.key,
		})
	}

	streamer := eng.(metaengine.StreamingScan)

	seq := streamer.StreamScan(ctx, "items", nil, nil)

	count := 0

	for val, err := range seq {
		if err != nil {
			t.Fatalf("StreamScan error: %v", err)
		}

		m, ok := val.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", val)
		}

		if m["name"] == nil {
			t.Error("name field is nil")
		}

		count++
	}

	if count != 4 {
		t.Errorf("expected 4 items, got %d", count)
	}
}

// TestPebbleStreamScan_WithFilter verifies filter pushdown in streaming mode.
func TestPebbleStreamScan_WithFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for _, item := range []struct {
		key    string
		status string
	}{
		{"a", "active"}, {"b", "inactive"}, {"c", "active"}, {"d", "archived"},
	} {
		_ = mb.MapSet(ctx, "users", item.key, map[string]any{
			"name":   item.key,
			"status": item.status,
		})
	}

	streamer := eng.(metaengine.StreamingScan)

	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	seq := streamer.StreamScan(ctx, "users", filters, nil)

	var names []string

	for val, err := range seq {
		if err != nil {
			t.Fatalf("StreamScan error: %v", err)
		}

		m := val.(map[string]any)
		if m["status"] != "active" {
			t.Errorf("expected status=active, got %v", m["status"])
		}

		names = append(names, m["name"].(string))
	}

	if len(names) != 2 {
		t.Errorf("expected 2 active users, got %d", len(names))
	}
}

// TestPebbleStreamScan_EarlyExit verifies that the consumer can stop iteration
// early and the iterator is properly cleaned up.
func TestPebbleStreamScan_EarlyExit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for i := range 100 {
		key := string(rune('a'+i%26)) + string(rune('a'+i/26))
		_ = mb.MapSet(ctx, "bulk", key, map[string]any{"idx": i})
	}

	streamer := eng.(metaengine.StreamingScan)

	seq := streamer.StreamScan(ctx, "bulk", nil, nil)

	count := 0

	for _, err := range seq {
		if err != nil {
			t.Fatalf("StreamScan error: %v", err)
		}

		count++

		if count >= 10 {
			break // Early exit after 10 items
		}
	}

	if count != 10 {
		t.Errorf("expected 10 items before early exit, got %d", count)
	}
}
