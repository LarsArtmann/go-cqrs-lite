package event

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

// streamingStore implements Store plus StreamingSource and StreamingJournal
// so DecorateStore's streaming forwarding can be exercised.
type streamingStore struct {
	fullStore
	streamErr error
}

func (s *streamingStore) LoadStream(
	_ context.Context,
	_ id.StreamRef,
) (EventIterator, error) {
	if s.streamErr != nil {
		return nil, s.streamErr
	}

	return NewSliceIterator(s.saved), nil
}

func (s *streamingStore) LoadStreamFromVersion(
	_ context.Context,
	_ id.StreamRef,
	_ Version,
) (EventIterator, error) {
	if s.streamErr != nil {
		return nil, s.streamErr
	}

	return NewSliceIterator(s.saved), nil
}

func (s *streamingStore) ReadStream(_ context.Context) (EventIterator, error) {
	if s.streamErr != nil {
		return nil, s.streamErr
	}

	return NewSliceIterator(s.saved), nil
}

func (s *streamingStore) ReadStreamFrom(
	_ context.Context,
	_ id.EventID,
	_ int,
) (EventIterator, error) {
	if s.streamErr != nil {
		return nil, s.streamErr
	}

	return NewSliceIterator(s.saved), nil
}

// TestDecorateStore_ForwardsStreamingCapabilities verifies that wrapping a
// streaming-capable store does not silently strip StreamingSource or
// StreamingJournal (the ADR-0126 wrapper-honesty contract).
func TestDecorateStore_ForwardsStreamingCapabilities(t *testing.T) {
	t.Parallel()

	inner := &streamingStore{}
	ctx := context.Background()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	for i := range 3 {
		evt, err := NewEvent("orders.created", streamID, "orders", Version(i+1), nil)
		if err != nil {
			t.Fatalf("NewEvent() error: %v", err)
		}

		inner.saved = append(inner.saved, evt)
	}

	store := DecorateStore(inner, nil, SourceTransform(passthrough))

	streamingSource, ok := store.(StreamingSource)
	if !ok {
		t.Fatal("DecorateStore must forward StreamingSource")
	}

	iter, err := streamingSource.LoadStream(ctx, testRef())
	if err != nil {
		t.Fatalf("LoadStream() error: %v", err)
	}

	count, err := drainIterator(iter)
	if err != nil {
		t.Fatalf("LoadStream() drain error: %v", err)
	}

	if count != len(inner.saved) {
		t.Errorf("LoadStream() yielded %d events, want %d", count, len(inner.saved))
	}

	streamingJournal, ok := store.(StreamingJournal)
	if !ok {
		t.Fatal("DecorateStore must forward StreamingJournal")
	}

	iter, err = streamingJournal.ReadStream(ctx)
	if err != nil {
		t.Fatalf("ReadStream() error: %v", err)
	}

	count, err = drainIterator(iter)
	if err != nil {
		t.Fatalf("ReadStream() drain error: %v", err)
	}

	if count != len(inner.saved) {
		t.Errorf("ReadStream() yielded %d events, want %d", count, len(inner.saved))
	}
}

// TestDecorateStore_SinkOnlyKeepsStreamingUnwrapped verifies that a
// sink-only decoration returns the inner iterator without wrapping.
func TestDecorateStore_SinkOnlyKeepsStreamingUnwrapped(t *testing.T) {
	t.Parallel()

	inner := &streamingStore{}
	store := DecorateStore(inner, SinkTransform(passthrough), nil)

	iter, err := store.(StreamingSource).LoadStream(context.Background(), testRef())
	if err != nil {
		t.Fatalf("LoadStream() error: %v", err)
	}

	if _, isSlice := iter.(*SliceIterator); !isSlice {
		t.Errorf("sink-only decoration must not wrap the iterator, got %T", iter)
	}

	if err := iter.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

// TestDecorateStore_InnerLacksStreaming verifies the rejection contract for
// inner stores without streaming support.
func TestDecorateStore_InnerLacksStreaming(t *testing.T) {
	t.Parallel()

	store := DecorateStore(&bareStore{}, nil, SourceTransform(passthrough))
	ctx := context.Background()

	if _, err := store.(StreamingSource).LoadStream(
		ctx,
		testRef(),
	); !errors.Is(err, ErrInnerStoreNotStreaming) {
		t.Errorf("LoadStream() error = %v, want ErrInnerStoreNotStreaming", err)
	}

	if _, err := store.(StreamingSource).LoadStreamFromVersion(
		ctx,
		testRef(),
		1,
	); !errors.Is(err, ErrInnerStoreNotStreaming) {
		t.Errorf("LoadStreamFromVersion() error = %v, want ErrInnerStoreNotStreaming", err)
	}

	if _, err := store.(StreamingJournal).ReadStream(
		ctx,
	); !errors.Is(err, ErrInnerStoreNotStreaming) {
		t.Errorf("ReadStream() error = %v, want ErrInnerStoreNotStreaming", err)
	}

	if _, err := store.(StreamingJournal).ReadStreamFrom(
		ctx,
		id.EventID{},
		1,
	); !errors.Is(err, ErrInnerStoreNotStreaming) {
		t.Errorf("ReadStreamFrom() error = %v, want ErrInnerStoreNotStreaming", err)
	}
}

func drainIterator(iter EventIterator) (int, error) {
	defer func() { _ = iter.Close() }()

	count := 0

	for {
		_, err := iter.Next()
		if errors.Is(err, io.EOF) {
			return count, nil
		}

		if err != nil {
			return count, err
		}

		count++
	}
}
