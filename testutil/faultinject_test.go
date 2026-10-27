package testutil_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

var errInjected = errors.New("injected backend failure")

// storeOnly implements event.Store and NOTHING else — the shape that catches a
// wrapper inventing capabilities.
type storeOnly struct {
	events []event.Event
}

func (s *storeOnly) Save(
	_ context.Context,
	_ id.StreamRef,
	events []event.Event,
	_ event.Version,
) error {
	s.events = append(s.events, events...)

	return nil
}

func (s *storeOnly) AppendBatch(_ context.Context, _ id.StreamRef, events []event.Event) error {
	s.events = append(s.events, events...)

	return nil
}

func (s *storeOnly) Load(_ context.Context, _ id.StreamRef) ([]event.Event, error) {
	return s.events, nil
}

func (s *storeOnly) LoadFromVersion(
	_ context.Context,
	_ id.StreamRef,
	_ event.Version,
) ([]event.Event, error) {
	return s.events, nil
}

func (s *storeOnly) LoadToVersion(
	_ context.Context,
	_ id.StreamRef,
	_ event.Version,
) ([]event.Event, error) {
	return s.events, nil
}

func (s *storeOnly) LoadToTimestamp(
	_ context.Context,
	_ id.StreamRef,
	_ time.Time,
) ([]event.Event, error) {
	return s.events, nil
}

// journalStore adds Journal + SeekableJournal + Close, i.e. everything
// eventtest.FakeStore provides today.
type journalStore struct {
	storeOnly

	closed bool
}

func (s *journalStore) ReadAll(_ context.Context) ([]event.Event, error) {
	return s.events, nil
}

func (s *journalStore) ReadFrom(
	_ context.Context,
	_ id.EventID,
	_ int,
) ([]event.Event, error) {
	return s.events, nil
}

func (s *journalStore) Close() error {
	s.closed = true

	return nil
}

func newEvent(t *testing.T, version event.Version) event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	evt, err := event.NewEvent("TestEvent", streamID, "Test", version, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func testRef() id.StreamRef {
	return id.NewStreamRef("Test", id.NewStreamID())
}

// TestFaultInjector_RuleFiresAtConfiguredCall pins the 1-based contract: At=2
// fails the SECOND call and nothing else. An off-by-one here would make a
// retry test assert on the wrong attempt.
func TestFaultInjector_RuleFiresAtConfiguredCall(t *testing.T) {
	t.Parallel()

	inner := &storeOnly{}
	injector := testutil.NewFaultInjector(inner, testutil.FaultRule{
		Method: testutil.MethodSave,
		At:     2,
		Err:    errInjected,
	})
	ctx := context.Background()
	ref := testRef()

	if err := injector.Save(ctx, ref, []event.Event{newEvent(t, 1)}, 0); err != nil {
		t.Fatalf("first Save must reach the inner store, got %v", err)
	}

	if err := injector.Save(
		ctx,
		ref,
		[]event.Event{newEvent(t, 2)},
		1,
	); !errors.Is(
		err,
		errInjected,
	) {
		t.Fatalf("second Save = %v, want the injected error", err)
	}

	if err := injector.Save(ctx, ref, []event.Event{newEvent(t, 3)}, 2); err != nil {
		t.Fatalf("third Save must recover, got %v", err)
	}

	if got := injector.Calls(testutil.MethodSave); got != 3 {
		t.Fatalf("Calls(Save) = %d, want 3", got)
	}

	if len(inner.events) != 2 {
		t.Fatalf(
			"inner store received %d events, want 2 (the failing call must not reach it)",
			len(inner.events),
		)
	}
}

// TestFaultInjector_AtZeroFailsEveryCall is the "backend is down" case used by
// outage tests.
func TestFaultInjector_AtZeroFailsEveryCall(t *testing.T) {
	t.Parallel()

	injector := testutil.NewFaultInjector(&storeOnly{}, testutil.FaultRule{
		Method: testutil.MethodLoad,
		At:     0,
		Err:    errInjected,
	})

	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := injector.Load(context.Background(), testRef()); !errors.Is(err, errInjected) {
			t.Fatalf("attempt %d: Load = %v, want the injected error", attempt, err)
		}
	}
}

// TestFaultInjector_OtherMethodsUnaffected proves the counter is per-method:
// a rule on Save must not perturb Load, and vice versa.
func TestFaultInjector_OtherMethodsUnaffected(t *testing.T) {
	t.Parallel()

	injector := testutil.NewFaultInjector(&storeOnly{}, testutil.FaultRule{
		Method: testutil.MethodSave,
		At:     1,
		Err:    errInjected,
	})
	ctx := context.Background()
	ref := testRef()

	if err := injector.Save(
		ctx,
		ref,
		[]event.Event{newEvent(t, 1)},
		0,
	); !errors.Is(
		err,
		errInjected,
	) {
		t.Fatalf("Save = %v, want the injected error", err)
	}

	if _, err := injector.Load(ctx, ref); err != nil {
		t.Fatalf("Load must be unaffected by a Save rule, got %v", err)
	}

	if got := injector.Calls(testutil.MethodLoad); got != 1 {
		t.Fatalf("Calls(Load) = %d, want 1", got)
	}
}

// TestFaultInjector_ForwardsCapabilities is the ADR-0126 regression guard: a
// wrapper must not drop MultiSink-style capabilities. Assertions alone are not
// enough (a lying implementation satisfies them), so each forwarded capability
// is also exercised and its call counted through the injector.
func TestFaultInjector_ForwardsCapabilities(t *testing.T) {
	t.Parallel()

	inner := &journalStore{}
	injector := testutil.NewFaultInjector(inner)
	ctx := context.Background()
	ref := testRef()

	if _, ok := any(injector).(event.Store); !ok {
		t.Fatal("injector must satisfy event.Store")
	}

	capabilities := injector.Capabilities()
	if !capabilities.Journal || !capabilities.SeekableJournal || !capabilities.Closer {
		t.Fatalf("capabilities = %+v, want Journal, SeekableJournal and Closer", capabilities)
	}

	if capabilities.MultiSink || capabilities.StreamingJournal || capabilities.BackwardsSource {
		t.Fatalf(
			"capabilities = %+v, want no MultiSink/Streaming/Backwards (the inner store has none)",
			capabilities,
		)
	}

	if _, err := injector.ReadAll(ctx); err != nil {
		t.Fatalf("ReadAll through the injector: %v", err)
	}

	if _, err := injector.ReadFrom(ctx, id.NewEventID(), 0); err != nil {
		t.Fatalf("ReadFrom through the injector: %v", err)
	}

	if err := injector.Close(); err != nil {
		t.Fatalf("Close through the injector: %v", err)
	}

	if !inner.closed {
		t.Fatal("Close did not reach the inner store")
	}

	if got := injector.Calls(testutil.MethodReadFrom); got != 1 {
		t.Fatalf("Calls(ReadFrom) = %d, want 1 (the injector must be the path, not a bypass)", got)
	}

	if _, err := injector.Load(ctx, ref); err != nil {
		t.Fatalf("Load through the injector: %v", err)
	}
}

// TestFaultInjector_MissingCapabilityIsLoud — the wrapper implements the method
// (Go has no conditional method sets), so a capability the inner lacks must be a
// typed error, never a silent success with zero values.
func TestFaultInjector_MissingCapabilityIsLoud(t *testing.T) {
	t.Parallel()

	injector := testutil.NewFaultInjector(&storeOnly{})
	ctx := context.Background()

	if _, err := injector.ReadAll(ctx); !errors.Is(err, testutil.ErrCapabilityUnavailable) {
		t.Fatalf("ReadAll on a store with no Journal = %v, want ErrCapabilityUnavailable", err)
	}

	if err := injector.SaveMultiBatch(
		ctx,
		nil,
	); !errors.Is(
		err,
		testutil.ErrCapabilityUnavailable,
	) {
		t.Fatalf(
			"SaveMultiBatch on a store with no MultiSink = %v, want ErrCapabilityUnavailable",
			err,
		)
	}

	if _, err := injector.ReadStream(ctx); !errors.Is(err, testutil.ErrCapabilityUnavailable) {
		t.Fatalf(
			"ReadStream on a store with no StreamingJournal = %v, want ErrCapabilityUnavailable",
			err,
		)
	}

	// The injector ALWAYS implements the interface (Go cannot build conditional
	// method sets), which is exactly why Capabilities() exists: the honesty
	// check is there, not in the type assertion.
	if _, ok := any(injector).(event.StreamingJournal); !ok {
		t.Fatal(
			"the injector stopped implementing StreamingJournal — the documented design is implemented-and-guarded",
		)
	}

	if injector.Capabilities().StreamingJournal {
		t.Fatal("Capabilities() claims StreamingJournal for an inner store that has none")
	}
}

// TestFaultInjector_PanicsOnBadRule — a rule that can never fire is a test that
// proves nothing, so the constructor refuses it instead of accepting it quietly.
func TestFaultInjector_PanicsOnBadRule(t *testing.T) {
	t.Parallel()

	cases := map[string]testutil.FaultRule{
		"unknown method": {Method: "Serialize", Err: errInjected},
		"nil error":      {Method: testutil.MethodSave},
		"negative At":    {Method: testutil.MethodSave, At: -1, Err: errInjected},
	}

	for name, rule := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Fatalf("NewFaultInjector accepted an invalid rule (%s)", name)
				}
			}()

			_ = testutil.NewFaultInjector(&storeOnly{}, rule)
		})
	}
}
