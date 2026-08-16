package event

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// fullJournal implements every journal capability: Journal,
// SeekableJournal, StreamingJournal, and io.Closer.
type fullJournal struct {
	events []Event
	closed bool
	readFn func() error
}

func (j *fullJournal) snapshot() ([]Event, error) {
	if j.readFn != nil {
		if err := j.readFn(); err != nil {
			return nil, err
		}
	}

	out := make([]Event, len(j.events))
	copy(out, j.events)

	return out, nil
}

func (j *fullJournal) ReadAll(context.Context) ([]Event, error) { return j.snapshot() }

func (j *fullJournal) ReadFrom(_ context.Context, _ id.EventID, _ int) ([]Event, error) {
	return j.snapshot()
}

func (j *fullJournal) ReadStream(context.Context) (EventIterator, error) {
	events, err := j.snapshot()
	if err != nil {
		return nil, err
	}

	return NewSliceIterator(events), nil
}

func (j *fullJournal) ReadStreamFrom(
	_ context.Context,
	_ id.EventID,
	_ int,
) (EventIterator, error) {
	return j.ReadStream(context.Background())
}

func (j *fullJournal) Close() error {
	j.closed = true

	return nil
}

// seekableOnlyJournal implements Journal + SeekableJournal but NOT
// StreamingJournal.
type seekableOnlyJournal struct {
	events []Event
}

func (j *seekableOnlyJournal) ReadAll(context.Context) ([]Event, error) {
	out := make([]Event, len(j.events)) //nolint:gocritic // test fake
	copy(out, j.events)

	return out, nil
}

func (j *seekableOnlyJournal) ReadFrom(
	context.Context,
	id.EventID,
	int,
) ([]Event, error) {
	return j.ReadAll(context.Background())
}

// readAllOnlyJournal implements only Journal.
type readAllOnlyJournal struct{ events []Event }

func (j *readAllOnlyJournal) ReadAll(context.Context) ([]Event, error) {
	out := make([]Event, len(j.events)) //nolint:gocritic // test fake
	copy(out, j.events)

	return out, nil
}

func drainIter(t *testing.T, iter EventIterator) []Event {
	t.Helper()

	var out []Event
	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatalf("Next() error: %v", err)
		}

		out = append(out, evt)
	}
}

// doubleTagged expands every event with one extra tagged event.
func doubleTagged(events []Event) ([]Event, error) {
	out := make([]Event, 0, len(events)*2)
	out = append(out, events...)

	for _, evt := range events {
		tagged, err := Single("user.tagged", evt.StreamID(), evt.StreamType(),
			Version(99), struct{}{})
		if err != nil {
			return nil, err
		}

		out = append(out, tagged...)
	}

	return out, nil
}

func TestDecorateJournal_NilTransform_ReturnsInner(t *testing.T) {
	t.Parallel()

	inner := &fullJournal{}
	got := DecorateJournal(inner, nil)
	if got != Journal(inner) {
		t.Fatal("DecorateJournal with nil transform must return the inner journal unchanged")
	}
}

func TestDecorateJournal_NilJournal_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("DecorateJournal(nil, ...) must panic")
		}
	}()

	_ = DecorateJournal(nil, passthrough) //nolint:staticcheck // testing the panic
}

func TestDecorateJournal_ReadAll_AppliesTransform(t *testing.T) {
	t.Parallel()

	inner := &fullJournal{events: testEvents(t)}
	journal := DecorateJournal(inner, SourceTransform(doubleTagged))

	got, err := journal.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	if len(got) != len(inner.events)*2 {
		t.Fatalf("ReadAll() returned %d events, want %d", len(got), len(inner.events)*2)
	}
}

func TestDecorateJournal_ReadFrom_AppliesTransform(t *testing.T) {
	t.Parallel()

	inner := &fullJournal{events: testEvents(t)}
	journal := DecorateJournal(inner, SourceTransform(doubleTagged))

	seekable, ok := journal.(SeekableJournal)
	if !ok {
		t.Fatal("decorated journal must implement SeekableJournal")
	}

	got, err := seekable.ReadFrom(context.Background(), id.EventID{}, 10)
	if err != nil {
		t.Fatalf("ReadFrom() error: %v", err)
	}

	if len(got) != len(inner.events)*2 {
		t.Fatalf("ReadFrom() returned %d events, want %d", len(got), len(inner.events)*2)
	}
}

func TestDecorateJournal_ReadFrom_InnerNotSeekable(t *testing.T) {
	t.Parallel()

	journal := DecorateJournal(&readAllOnlyJournal{}, passthrough)

	seekable, ok := journal.(SeekableJournal)
	if !ok {
		t.Fatal("decorated journal must implement SeekableJournal")
	}

	_, err := seekable.ReadFrom(context.Background(), id.EventID{}, 10)
	if !errors.Is(err, ErrInnerStoreNotSeekable) {
		t.Fatalf("ReadFrom() error = %v, want ErrInnerStoreNotSeekable", err)
	}
}

func TestDecorateJournal_ReadStream_AppliesTransformAcrossChunks(t *testing.T) {
	t.Parallel()

	const total = journalTransformChunkSize*2 + 7
	inner := &fullJournal{events: make([]Event, 0, total)}
	for len(inner.events) < total {
		inner.events = append(inner.events, testEvents(t)...)
	}

	journal := DecorateJournal(inner, SourceTransform(doubleTagged))

	streaming, ok := journal.(StreamingJournal)
	if !ok {
		t.Fatal("decorated journal must implement StreamingJournal")
	}

	iter, err := streaming.ReadStream(context.Background())
	if err != nil {
		t.Fatalf("ReadStream() error: %v", err)
	}
	defer iter.Close()

	got := drainIter(t, iter)
	if len(got) != total*2 {
		t.Fatalf("ReadStream() yielded %d events, want %d across chunk boundaries", len(got), total*2)
	}
}

func TestDecorateJournal_ReadStreamFrom_FilteringTransform(t *testing.T) {
	t.Parallel()

	inner := &fullJournal{events: make([]Event, 0, 8)}
	for len(inner.events) < 8 {
		inner.events = append(inner.events, testEvents(t)...)
	}

	keepEven := func(events []Event) ([]Event, error) {
		out := make([]Event, 0, len(events)/2)
		for i, evt := range events {
			if i%2 == 0 {
				out = append(out, evt)
			}
		}

		return out, nil
	}

	journal := DecorateJournal(inner, keepEven)

	streaming := journal.(StreamingJournal)
	iter, err := streaming.ReadStreamFrom(context.Background(), id.EventID{}, 0)
	if err != nil {
		t.Fatalf("ReadStreamFrom() error: %v", err)
	}
	defer iter.Close()

	if got := drainIter(t, iter); len(got) != 4 {
		t.Fatalf("ReadStreamFrom() yielded %d events, want 4 after filtering", len(got))
	}
}

func TestDecorateJournal_ReadStream_InnerNotStreaming(t *testing.T) {
	t.Parallel()

	journal := DecorateJournal(&seekableOnlyJournal{}, passthrough)

	streaming, ok := journal.(StreamingJournal)
	if !ok {
		t.Fatal("decorated journal must implement StreamingJournal")
	}

	_, err := streaming.ReadStream(context.Background())
	if !errors.Is(err, ErrInnerStoreNotStreaming) {
		t.Fatalf("ReadStream() error = %v, want ErrInnerStoreNotStreaming", err)
	}
}

func TestDecorateJournal_StreamTransformError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("transform boom")
	journal := DecorateJournal(&fullJournal{events: testEvents(t)}, func([]Event) ([]Event, error) {
		return nil, wantErr
	})

	iter, err := journal.(StreamingJournal).ReadStream(context.Background())
	if err != nil {
		t.Fatalf("ReadStream() error: %v", err)
	}
	defer iter.Close()

	if _, err := iter.Next(); !errors.Is(err, wantErr) {
		t.Fatalf("Next() error = %v, want transform error", err)
	}
}

func TestDecorateJournal_ReadErrorPassthrough(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read boom")
	called := false
	journal := DecorateJournal(&fullJournal{readFn: func() error { return readErr }},
		func(events []Event) ([]Event, error) {
			called = true

			return events, nil
		})

	if _, err := journal.ReadAll(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("ReadAll() error = %v, want inner read error", err)
	}

	if called {
		t.Fatal("source transform must not run when the inner read failed")
	}
}

func TestDecorateJournal_Close_Delegates(t *testing.T) {
	t.Parallel()

	inner := &fullJournal{}
	journal := DecorateJournal(inner, passthrough)

	if err := journal.(interface{ Close() error }).Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if !inner.closed {
		t.Fatal("Close() must close the inner journal")
	}
}

func TestDecorateJournal_CloseWithoutCloser_IsNoOp(t *testing.T) {
	t.Parallel()

	journal := DecorateJournal(&readAllOnlyJournal{}, passthrough)

	if err := journal.(interface{ Close() error }).Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil for inner without io.Closer", err)
	}
}

func TestTransformingIterator_CloseThenNext(t *testing.T) {
	t.Parallel()

	inner := NewSliceIterator(testEvents(t))
	iter := &transformingIterator{inner: inner, sourceT: passthrough}

	if err := iter.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if _, err := iter.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("Next() after Close() = %v, want io.EOF", err)
	}
}
