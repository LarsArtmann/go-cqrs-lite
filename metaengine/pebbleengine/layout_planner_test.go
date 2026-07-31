package pebbleengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

func TestPebbleLayoutPlanner_SecondaryIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("expected pebbleEngine to implement LayoutPlanner")
	}

	// Declare a layout plan for "users" with "status" as a filter field.
	if err := lp.ApplyLayout("users", []string{"status"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write 5 users: 3 active, 2 inactive.
	users := []struct {
		key    string
		status string
		name   string
	}{
		{"u1", "active", "Alice"},
		{"u2", "active", "Bob"},
		{"u3", "inactive", "Charlie"},
		{"u4", "active", "Diana"},
		{"u5", "inactive", "Eve"},
	}

	for _, u := range users {
		if err := mb.MapSet(ctx, "users", u.key, map[string]any{
			"status": u.status,
			"name":   u.name,
		}); err != nil {
			t.Fatalf("MapSet %s: %v", u.key, err)
		}
	}

	// Scan with filter on "status" = "active" — should use the index.
	rawReader, ok := eng.(metaengine.RawScanReader)
	if !ok {
		t.Fatal("expected pebbleEngine to implement RawScanReader")
	}

	results, err := rawReader.ScanRawValues(ctx, "users",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "active"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 active users, got %d", len(results))
	}

	// Verify all results have status = "active".
	for _, raw := range results {
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		if decoded["status"] != "active" {
			t.Errorf("expected status 'active', got %v", decoded["status"])
		}
	}
}

func TestPebbleLayoutPlanner_UpdateReindexes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("items", []string{"cat"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write with category "a".
	_ = mb.MapSet(ctx, "items", "k1", map[string]any{"cat": "a", "val": 1})

	// Update to category "b".
	_ = mb.MapSet(ctx, "items", "k1", map[string]any{"cat": "b", "val": 2})

	// Scan for cat="a" — should return 0 (old index removed).
	rawReader := eng.(metaengine.RawScanReader)

	resultsA, err := rawReader.ScanRawValues(ctx, "items",
		[]metaengine.FilterSpec{{Column: "cat", Op: metaengine.FilterEq, Value: "a"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues cat=a: %v", err)
	}

	if len(resultsA) != 0 {
		t.Errorf("expected 0 items with cat=a after update, got %d", len(resultsA))
	}

	// Scan for cat="b" — should return 1.
	resultsB, err := rawReader.ScanRawValues(ctx, "items",
		[]metaengine.FilterSpec{{Column: "cat", Op: metaengine.FilterEq, Value: "b"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues cat=b: %v", err)
	}

	if len(resultsB) != 1 {
		t.Errorf("expected 1 item with cat=b after update, got %d", len(resultsB))
	}
}
