package view

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// TestBatchSet_LargeValuesChunkSafely proves the statement byte cap splits
// batches whose view values alone exceed it: six 2 MiB names cannot share one
// statement (12 MiB > sql.MaxStatementBytes), so BatchSet must chunk while
// still writing every row exactly once.
func TestBatchSet_LargeValuesChunkSafely(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	bigName := strings.Repeat("x", 2<<20)
	const itemCount = 6

	items := make([]kv.ViewItem[testView, testKey], itemCount)
	for i := range items {
		items[i] = kv.ViewItem[testView, testKey]{
			Key: testKey(strings.Repeat("k", i+1)),
			Value: &testView{
				Name:  bigName,
				Email: "big@example.com",
				Age:   i,
			},
		}
	}

	if err := store.BatchSet(ctx, items); err != nil {
		t.Fatalf("BatchSet: %v", err)
	}

	for i := range items {
		got, err := store.Get(ctx, items[i].Key)
		if err != nil {
			t.Fatalf("Get %s: %v", items[i].Key, err)
		}
		if len(got.Name) != 2<<20 {
			t.Errorf("key %s: name length = %d, want %d", items[i].Key, len(got.Name), 2<<20)
		}
		if got.Age != i {
			t.Errorf("key %s: age = %d, want %d", items[i].Key, got.Age, i)
		}
	}
}

func TestEstimateArgBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  any
		min  int
	}{
		{"string scales with length", strings.Repeat("a", 5000), 5000},
		{"bytes scale with length", make([]byte, 3000), 3000},
		{"time gets fixed slack", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 30},
	}

	for _, tt := range tests {
		if got := estimateArgBytes(tt.arg); got < tt.min {
			t.Errorf("%s: estimateArgBytes = %d, want >= %d", tt.name, got, tt.min)
		}
	}

	if got := estimateArgBytes(42); got != 32 {
		t.Errorf("numeric: estimateArgBytes = %d, want 32", got)
	}
}
