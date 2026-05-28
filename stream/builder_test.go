package stream_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stream"
)

func TestListBuilder_Chaining(t *testing.T) {
	t.Parallel()

	b := stream.NewListBuilder(&stubReader{})

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
		listFn: func(_ context.Context, opts stream.ListOptions) (*stream.Page[stream.AggregateRef], error) {
			called = true

			if opts.Limit != 20 {
				t.Errorf("default limit = %d, want 20", opts.Limit)
			}

			if opts.Tombstone != stream.TombstoneExclude {
				t.Errorf("default tombstone = %v, want TombstoneExclude", opts.Tombstone)
			}

			return &stream.Page[stream.AggregateRef]{}, nil
		},
	}

	_, err := stream.NewListBuilder(reader).OfType("User").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Error("expected List to be called")
	}
}

type stubReader struct {
	listFn           func(ctx context.Context, opts stream.ListOptions) (*stream.Page[stream.AggregateRef], error)
	listWithStatusFn func(ctx context.Context, opts stream.ListOptions) (*stream.Page[stream.AggregateStatus], error)
}

func (s *stubReader) List(ctx context.Context, opts stream.ListOptions) (*stream.Page[stream.AggregateRef], error) {
	if s.listFn != nil {
		return s.listFn(ctx, opts)
	}

	return &stream.Page[stream.AggregateRef]{}, nil
}

func (s *stubReader) ListWithStatus(
	ctx context.Context,
	opts stream.ListOptions,
) (*stream.Page[stream.AggregateStatus], error) {
	if s.listWithStatusFn != nil {
		return s.listWithStatusFn(ctx, opts)
	}

	return &stream.Page[stream.AggregateStatus]{}, nil
}
