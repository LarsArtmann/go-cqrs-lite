package pebbleengine

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/onsi/gomega"
)

func TestPebbleScanCount_NoFilters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for i := range 50 {
		_ = mb.MapSet(ctx, "items", string(rune('a'+i)), map[string]any{"v": i})
	}

	counter := eng.(ScanCounter)

	count, err := counter.ScanCount(ctx, "items", nil)
	if err != nil {
		t.Fatalf("ScanCount: %v", err)
	}

	if count != 50 {
		t.Errorf("expected 50, got %d", count)
	}
}

func TestPebbleScanCount_WithFilters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)

	for i := range 100 {
		status := "active"
		if i%3 == 0 {
			status = "inactive"
		}

		_ = mb.MapSet(ctx, "users", string(rune('a'+i%26))+string(rune('a'+i/26)),
			map[string]any{"status": status})
	}

	counter := eng.(ScanCounter)

	count, err := counter.ScanCount(ctx, "users", []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	})
	if err != nil {
		t.Fatalf("ScanCount: %v", err)
	}

	// i%3 != 0 → 100 - 34 = 66 active items (0,3,6,...,99 are inactive = 34 items)
	if count != 66 {
		t.Errorf("expected 66 active users, got %d", count)
	}
}
