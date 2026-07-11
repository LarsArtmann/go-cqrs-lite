package listing_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

func TestListBuilder_Chaining(t *testing.T) {
	t.Parallel()

	b := listing.NewListBuilder(&stubReader{})

	if b.OfType("User") != b {
		t.Error("OfType should return builder")
	}

	if b.PageSize(10) != b {
		t.Error("PageSize should return builder")
	}

	if b.IncludeDeleted() != b {
		t.Error("IncludeDeleted should return builder")
	}

	if b.OnlyDeleted() != b {
		t.Error("OnlyDeleted should return builder")
	}
}

func TestListBuilder_DefaultOptions(t *testing.T) {
	t.Parallel()

	called := false
	reader := &stubReader{
		listFn: func(_ context.Context, opts listing.ListOptions) (*listing.Page[listing.AggregateListing], error) {
			called = true

			if opts.Limit != 20 {
				t.Errorf("default limit = %d, want 20", opts.Limit)
			}

			if opts.Tombstone != listing.TombstoneExclude {
				t.Errorf("default tombstone = %v, want TombstoneExclude", opts.Tombstone)
			}

			return &listing.Page[listing.AggregateListing]{}, nil
		},
	}

	_, err := listing.NewListBuilder(reader).OfType("User").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Error("expected List to be called")
	}
}

type stubReader struct {
	listFn           func(ctx context.Context, opts listing.ListOptions) (*listing.Page[listing.AggregateListing], error)
	listWithStatusFn func(ctx context.Context, opts listing.ListOptions) (*listing.Page[listing.AggregateStatus], error)
}

func (s *stubReader) List(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.AggregateListing], error) {
	if s.listFn != nil {
		return s.listFn(ctx, opts)
	}

	return &listing.Page[listing.AggregateListing]{}, nil
}

func (s *stubReader) ListWithStatus(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.AggregateStatus], error) {
	if s.listWithStatusFn != nil {
		return s.listWithStatusFn(ctx, opts)
	}

	return &listing.Page[listing.AggregateStatus]{}, nil
}

func TestTombstonePolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		policy listing.TombstonePolicy
		want   string
	}{
		{listing.TombstoneExclude, "exclude"},
		{listing.TombstoneInclude, "include"},
		{listing.TombstoneOnly, "only"},
		{listing.TombstonePolicy(99), "TombstonePolicy(99)"},
	}

	for _, tt := range tests {
		got := tt.policy.String()
		if got != tt.want {
			t.Errorf("TombstonePolicy(%d).String() = %q, want %q", tt.policy, got, tt.want)
		}
	}
}

func TestAggregateStatus_MarshalJSON(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	aggID := id.NewAggregateID()

	status := listing.AggregateStatus{
		Ref: listing.AggregateListing{
			ID:          aggID,
			Type:        id.AggregateType("User"),
			Version:     event.Version(3),
			EventCount:  3,
			LastEventAt: ts,
		},
		Status: event.TombstoneActive,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}

	if result["type"] != "User" {
		t.Errorf("type = %v, want User", result["type"])
	}

	if result["status"] != "active" {
		t.Errorf("status = %v, want active", result["status"])
	}

	if result["version"] != float64(3) {
		t.Errorf("version = %v, want 3", result["version"])
	}

	if result["event_count"] != float64(3) {
		t.Errorf("event_count = %v, want 3", result["event_count"])
	}
}
