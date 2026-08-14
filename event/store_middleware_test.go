package event

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// fullStore implements every store capability: Store, Journal,
// SeekableJournal, BackwardsSource, MultiSink, and io.Closer.
type fullStore struct {
	saved      []Event
	multiSaved int
	closed     bool
	readErr    error
}

func (s *fullStore) Save(
	_ context.Context,
	_ id.StreamRef,
	events []Event,
	_ Version,
) error {
	s.saved = append(s.saved, events...)
	return nil
}

func (s *fullStore) AppendBatch(
	_ context.Context,
	_ id.StreamRef,
	events []Event,
) error {
	s.saved = append(s.saved, events...)
	return nil
}

func (s *fullStore) SaveMultiBatch(_ context.Context, _ []MultiBatchEntry) error {
	s.multiSaved++
	return nil
}

func (s *fullStore) snapshot(events []Event) ([]Event, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}

	out := make([]Event, len(events))
	copy(out, events)
	return out, nil
}

func (s *fullStore) Load(_ context.Context, _ id.StreamRef) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) LoadFromVersion(
	_ context.Context,
	_ id.StreamRef,
	_ Version,
) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) LoadToVersion(
	_ context.Context,
	_ id.StreamRef,
	_ Version,
) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) LoadToTimestamp(
	_ context.Context,
	_ id.StreamRef,
	_ time.Time,
) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) ReadAll(_ context.Context) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) ReadFrom(
	_ context.Context,
	_ id.EventID,
	_ int,
) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) LoadBackwards(_ context.Context, _ id.StreamRef) ([]Event, error) {
	return s.snapshot(s.saved)
}

func (s *fullStore) Close() error {
	s.closed = true
	return nil
}

// bareStore implements only Store — no optional capabilities.
type bareStore struct{ saved []Event }

func (s *bareStore) Save(
	_ context.Context,
	_ id.StreamRef,
	events []Event,
	_ Version,
) error {
	s.saved = append(s.saved, events...)
	return nil
}

func (s *bareStore) AppendBatch(_ context.Context, _ id.StreamRef, _ []Event) error {
	return nil
}

func (s *bareStore) Load(_ context.Context, _ id.StreamRef) ([]Event, error) {
	out := make([]Event, len(s.saved))
	copy(out, s.saved)
	return out, nil
}

func (s *bareStore) LoadFromVersion(
	_ context.Context,
	_ id.StreamRef,
	_ Version,
) ([]Event, error) {
	return s.Load(context.Background(), id.StreamRef{})
}

func (s *bareStore) LoadToVersion(
	_ context.Context,
	_ id.StreamRef,
	_ Version,
) ([]Event, error) {
	return s.Load(context.Background(), id.StreamRef{})
}

func (s *bareStore) LoadToTimestamp(
	_ context.Context,
	_ id.StreamRef,
	_ time.Time,
) ([]Event, error) {
	return s.Load(context.Background(), id.StreamRef{})
}

func testEvents(t *testing.T) []Event {
	t.Helper()

	streamID := id.NewStreamID()
	streamType := id.StreamType("User")

	events, err := Single("user.created", streamID, streamType, Version(1),
		struct{ Name string }{Name: "Alice"})
	if err != nil {
		t.Fatalf("Single() error: %v", err)
	}

	return events
}

// passthrough is a no-op transform that forces DecorateStore onto the
// wrapper path (both-nil returns the inner store unchanged).
func passthrough(events []Event) ([]Event, error) { return events, nil }

// testRef returns a fresh stream reference.
func testRef() id.StreamRef {
	return id.NewStreamRef(id.StreamType("User"), id.NewStreamID())
}

func TestDecorateStore_NilTransforms_ReturnsInner(t *testing.T) {
	t.Parallel()

	inner := &fullStore{}
	got := DecorateStore(inner, nil, nil)
	if got != Store(inner) {
		t.Fatal("DecorateStore with both transforms nil must return the inner store unchanged")
	}
}

func TestDecorateStore_NilStore_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("DecorateStore(nil, ...) must panic")
		}
	}()

	_ = DecorateStore(nil, nil, nil) //nolint:staticcheck // testing the panic
}

func TestDecorateStore_SinkTransform(t *testing.T) {
	t.Parallel()

	inner := &fullStore{}
	store := DecorateStore(inner, SinkTransform(passthrough), nil)

	if err := store.Save(context.Background(), testRef(), testEvents(t), Version(1)); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if len(inner.saved) != 1 {
		t.Fatalf("inner saved %d events, want 1", len(inner.saved))
	}
}

func TestDecorateStore_SinkTransformError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("sink boom")
	inner := &fullStore{}
	store := DecorateStore(inner, func([]Event) ([]Event, error) {
		return nil, wantErr
	}, nil)

	err := store.Save(context.Background(), testRef(), testEvents(t), Version(1))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Save() error = %v, want sink error", err)
	}

	if len(inner.saved) != 0 {
		t.Fatalf("inner saved %d events, want 0 on transform error", len(inner.saved))
	}
}

func TestDecorateStore_SourceTransform(t *testing.T) {
	t.Parallel()

	saved := testEvents(t)
	inner := &fullStore{}
	inner.saved = append(inner.saved, saved...)

	marker := func(events []Event) ([]Event, error) {
		if len(events) == 0 {
			return events, nil
		}

		tagged, err := Single("user.tagged", events[0].StreamID(), events[0].StreamType(),
			Version(99), struct{}{})
		if err != nil {
			return nil, err
		}

		return append(events, tagged...), nil
	}
	store := DecorateStore(inner, nil, SourceTransform(marker))

	ctx := context.Background()
	ref := testRef()

	reads := map[string]func() ([]Event, error){
		"Load":            func() ([]Event, error) { return store.Load(ctx, ref) },
		"LoadFromVersion": func() ([]Event, error) { return store.LoadFromVersion(ctx, ref, Version(1)) },
		"LoadToVersion":   func() ([]Event, error) { return store.LoadToVersion(ctx, ref, Version(1)) },
		"LoadToTimestamp": func() ([]Event, error) { return store.LoadToTimestamp(ctx, ref, time.Now()) },
		"ReadAll":         func() ([]Event, error) { return store.(Journal).ReadAll(ctx) },
		"ReadFrom":        func() ([]Event, error) { return store.(SeekableJournal).ReadFrom(ctx, id.EventID{}, 10) },
		"LoadBackwards":   func() ([]Event, error) { return store.(BackwardsSource).LoadBackwards(ctx, ref) },
	}

	for name, read := range reads {
		got, err := read()
		if err != nil {
			t.Fatalf("%s() error: %v", name, err)
		}

		if len(got) != len(saved)+1 {
			t.Errorf("%s() returned %d events, want %d (source transform applied)", name,
				len(got), len(saved)+1)
		}
	}
}

func TestDecorateStore_SourceTransformReadErrorPassthrough(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read boom")
	inner := &fullStore{readErr: readErr}
	called := false
	store := DecorateStore(inner, nil, func(events []Event) ([]Event, error) {
		called = true
		return events, nil
	})

	if _, err := store.Load(context.Background(), testRef()); !errors.Is(err, readErr) {
		t.Fatalf("Load() error = %v, want inner read error", err)
	}

	if called {
		t.Fatal("source transform must not run when the inner read failed")
	}
}

func TestDecorateStore_MultiSink_AppliesSinkTransform(t *testing.T) {
	t.Parallel()

	inner := &fullStore{}
	wantErr := errors.New("sink boom")
	store := DecorateStore(inner, func([]Event) ([]Event, error) {
		return nil, wantErr
	}, nil)

	multi, ok := store.(MultiSink)
	if !ok {
		t.Fatal("decorated store must implement MultiSink")
	}

	err := multi.SaveMultiBatch(context.Background(),
		[]MultiBatchEntry{{Ref: testRef(), Events: testEvents(t)}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("SaveMultiBatch() error = %v, want sink error", err)
	}

	if inner.multiSaved != 0 {
		t.Fatalf("inner SaveMultiBatch called %d times, want 0 on transform error",
			inner.multiSaved)
	}
}

func TestDecorateStore_Close_Delegates(t *testing.T) {
	t.Parallel()

	inner := &fullStore{}
	store := DecorateStore(inner, nil, SourceTransform(passthrough))

	if err := store.(io.Closer).Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if !inner.closed {
		t.Fatal("Close() must close the inner store")
	}
}

func TestDecorateStore_InnerLacksOptionalCapabilities(t *testing.T) {
	t.Parallel()

	store := DecorateStore(&bareStore{}, nil, SourceTransform(passthrough))
	ctx := context.Background()

	if _, err := store.(Journal).ReadAll(ctx); !errors.Is(err, ErrInnerStoreNotJournal) {
		t.Errorf("ReadAll() error = %v, want ErrInnerStoreNotJournal", err)
	}

	if _, err := store.(SeekableJournal).ReadFrom(
		ctx,
		id.EventID{},
		1,
	); !errors.Is(
		err,
		ErrInnerStoreNotSeekable,
	) {
		t.Errorf("ReadFrom() error = %v, want ErrInnerStoreNotSeekable", err)
	}

	if _, err := store.(BackwardsSource).LoadBackwards(
		ctx,
		testRef(),
	); !errors.Is(
		err,
		ErrInnerStoreNotBackwards,
	) {
		t.Errorf("LoadBackwards() error = %v, want ErrInnerStoreNotBackwards", err)
	}

	err := store.(MultiSink).SaveMultiBatch(ctx, nil)
	if !errors.Is(err, ErrInnerStoreNotMultiSink) {
		t.Errorf("SaveMultiBatch() error = %v, want ErrInnerStoreNotMultiSink", err)
	}
}

func TestDecorateStore_CloseWithoutCloser_IsNoOp(t *testing.T) {
	t.Parallel()

	store := DecorateStore(&bareStore{}, nil, SourceTransform(passthrough))

	if err := store.(io.Closer).Close(); err != nil {
		t.Fatalf("Close() on non-closer inner store = %v, want nil", err)
	}
}
