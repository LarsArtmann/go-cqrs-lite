package testutil

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ErrCapabilityUnavailable is returned when a caller uses an optional store
// capability that the wrapped store does not implement. It exists so a
// missing capability is a loud, typed failure instead of a silent one.
var ErrCapabilityUnavailable = errors.New(
	"testutil: wrapped store does not implement this capability",
)

// The method names accepted by FaultRule.Method. Strings (not constants of a
// custom type) keep the call sites readable and let a typo fail at
// construction: NewFaultInjector panics on an unknown name, because a rule that
// never fires is a test that silently proves nothing.
const (
	MethodSave            = "Save"
	MethodAppendBatch     = "AppendBatch"
	MethodLoad            = "Load"
	MethodLoadFromVersion = "LoadFromVersion"
	MethodLoadToVersion   = "LoadToVersion"
	MethodLoadToTimestamp = "LoadToTimestamp"
	MethodSaveMultiBatch  = "SaveMultiBatch"
	MethodReadAll         = "ReadAll"
	MethodReadFrom        = "ReadFrom"
	MethodReadStream      = "ReadStream"
	MethodReadStreamFrom  = "ReadStreamFrom"
	MethodLoadBackwards   = "LoadBackwards"
	MethodClose           = "Close"
)

var knownMethods = map[string]bool{
	MethodSave:            true,
	MethodAppendBatch:     true,
	MethodLoad:            true,
	MethodLoadFromVersion: true,
	MethodLoadToVersion:   true,
	MethodLoadToTimestamp: true,
	MethodSaveMultiBatch:  true,
	MethodReadAll:         true,
	MethodReadFrom:        true,
	MethodReadStream:      true,
	MethodReadStreamFrom:  true,
	MethodLoadBackwards:   true,
	MethodClose:           true,
}

// FaultRule injects Err instead of calling the inner store at one point in a
// call sequence.
//
// At is 1-based: At=1 fails the first call of that method, At=2 the second.
// At=0 fails EVERY call (the "the backend is down" case). Calls of other
// methods do not advance the counter, so a rule survives unrelated traffic.
type FaultRule struct {
	Method string
	At     int
	Err    error
}

// Capabilities records which optional interfaces the WRAPPED store implements.
// The injector itself implements all of them (Go has no dynamic method sets),
// so this is how a test asserts "the wrapper did not invent a capability".
type Capabilities struct {
	MultiSink        bool
	Journal          bool
	SeekableJournal  bool
	StreamingJournal bool
	BackwardsSource  bool
	Closer           bool
}

// FaultInjector wraps an event.Store, fails configured calls, and forwards
// everything else — including optional capabilities — to the wrapped store.
//
// Why a shared helper: ADR-0126 exists because hand-written store wrappers kept
// silently dropping capabilities (the old encryptedStore lost MultiSink; the old
// VersionedSeekableJournal lost StreamingJournal). Any test double that wraps a
// store can reintroduce that bug, so the wrapper is written once, forwards every
// capability, and reports which ones it actually has.
type FaultInjector struct {
	inner event.Store
	rules []FaultRule

	mu    sync.Mutex
	calls map[string]int
}

// NewFaultInjector wraps inner. It panics on an unknown method name or a nil
// error: both are authoring mistakes that would make a test vacuous.
func NewFaultInjector(inner event.Store, rules ...FaultRule) *FaultInjector {
	if inner == nil {
		panic("testutil: NewFaultInjector called with a nil store")
	}

	for _, rule := range rules {
		if !knownMethods[rule.Method] {
			panic(fmt.Sprintf("testutil: FaultRule.Method %q is not a store method", rule.Method))
		}

		if rule.Err == nil {
			panic(fmt.Sprintf("testutil: FaultRule for %s has a nil error", rule.Method))
		}

		if rule.At < 0 {
			panic(
				fmt.Sprintf(
					"testutil: FaultRule for %s has a negative At (%d)",
					rule.Method,
					rule.At,
				),
			)
		}
	}

	return &FaultInjector{
		inner: inner,
		rules: rules,
		calls: map[string]int{},
	}
}

// Calls reports how many times method was invoked through the injector.
func (f *FaultInjector) Calls(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[method]
}

// Capabilities reports the wrapped store's actual capability set.
func (f *FaultInjector) Capabilities() Capabilities {
	var capabilities Capabilities

	if _, ok := f.inner.(event.MultiSink); ok {
		capabilities.MultiSink = true
	}

	if _, ok := f.inner.(event.Journal); ok {
		capabilities.Journal = true
	}

	if _, ok := f.inner.(event.SeekableJournal); ok {
		capabilities.SeekableJournal = true
	}

	if _, ok := f.inner.(event.StreamingJournal); ok {
		capabilities.StreamingJournal = true
	}

	if _, ok := f.inner.(event.BackwardsSource); ok {
		capabilities.BackwardsSource = true
	}

	if _, ok := f.inner.(interface{ Close() error }); ok {
		capabilities.Closer = true
	}

	return capabilities
}

// arm advances the call counter for method and returns the rule's error when
// this call is the one that must fail.
func (f *FaultInjector) arm(method string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls[method]++
	call := f.calls[method]

	for _, rule := range f.rules {
		if rule.Method != method {
			continue
		}

		if rule.At == 0 || rule.At == call {
			return rule.Err
		}
	}

	return nil
}

// unavailable builds the typed error for a capability the inner store lacks.
func unavailable(capability string) error {
	return fmt.Errorf("%w: %s", ErrCapabilityUnavailable, capability)
}

// Save appends events with an optimistic concurrency check.
func (f *FaultInjector) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if err := f.arm(MethodSave); err != nil {
		return err
	}

	return f.inner.Save(ctx, ref, events, expectedVersion)
}

// AppendBatch appends without a concurrency check.
func (f *FaultInjector) AppendBatch(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
) error {
	if err := f.arm(MethodAppendBatch); err != nil {
		return err
	}

	return f.inner.AppendBatch(ctx, ref, events)
}

// Load retrieves every event of a stream.
func (f *FaultInjector) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	if err := f.arm(MethodLoad); err != nil {
		return nil, err
	}

	return f.inner.Load(ctx, ref)
}

// LoadFromVersion retrieves events after version (exclusive).
func (f *FaultInjector) LoadFromVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) ([]event.Event, error) {
	if err := f.arm(MethodLoadFromVersion); err != nil {
		return nil, err
	}

	return f.inner.LoadFromVersion(ctx, ref, version)
}

// LoadToVersion retrieves events up to and including maxVersion.
func (f *FaultInjector) LoadToVersion(
	ctx context.Context,
	ref id.StreamRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	if err := f.arm(MethodLoadToVersion); err != nil {
		return nil, err
	}

	return f.inner.LoadToVersion(ctx, ref, maxVersion)
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
func (f *FaultInjector) LoadToTimestamp(
	ctx context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]event.Event, error) {
	if err := f.arm(MethodLoadToTimestamp); err != nil {
		return nil, err
	}

	return f.inner.LoadToTimestamp(ctx, ref, maxTime)
}

// SaveMultiBatch forwards to the inner MultiSink, or fails loudly when the
// wrapped store has none.
func (f *FaultInjector) SaveMultiBatch(ctx context.Context, entries []event.MultiBatchEntry) error {
	if err := f.arm(MethodSaveMultiBatch); err != nil {
		return err
	}

	multi, ok := f.inner.(event.MultiSink)
	if !ok {
		return unavailable(MethodSaveMultiBatch)
	}

	return multi.SaveMultiBatch(ctx, entries)
}

// ReadAll forwards to the inner Journal.
func (f *FaultInjector) ReadAll(ctx context.Context) ([]event.Event, error) {
	if err := f.arm(MethodReadAll); err != nil {
		return nil, err
	}

	journal, ok := f.inner.(event.Journal)
	if !ok {
		return nil, unavailable(MethodReadAll)
	}

	return journal.ReadAll(ctx)
}

// ReadFrom forwards to the inner SeekableJournal.
func (f *FaultInjector) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if err := f.arm(MethodReadFrom); err != nil {
		return nil, err
	}

	seekable, ok := f.inner.(event.SeekableJournal)
	if !ok {
		return nil, unavailable(MethodReadFrom)
	}

	return seekable.ReadFrom(ctx, afterEventID, limit)
}

// ReadStream forwards to the inner StreamingJournal.
func (f *FaultInjector) ReadStream(ctx context.Context) (event.EventIterator, error) {
	if err := f.arm(MethodReadStream); err != nil {
		return nil, err
	}

	streaming, ok := f.inner.(event.StreamingJournal)
	if !ok {
		return nil, unavailable(MethodReadStream)
	}

	return streaming.ReadStream(ctx)
}

// ReadStreamFrom forwards to the inner StreamingJournal.
func (f *FaultInjector) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	if err := f.arm(MethodReadStreamFrom); err != nil {
		return nil, err
	}

	streaming, ok := f.inner.(event.StreamingJournal)
	if !ok {
		return nil, unavailable(MethodReadStreamFrom)
	}

	return streaming.ReadStreamFrom(ctx, afterEventID, limit)
}

// LoadBackwards forwards to the inner BackwardsSource.
func (f *FaultInjector) LoadBackwards(
	ctx context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	if err := f.arm(MethodLoadBackwards); err != nil {
		return nil, err
	}

	backwards, ok := f.inner.(event.BackwardsSource)
	if !ok {
		return nil, unavailable(MethodLoadBackwards)
	}

	return backwards.LoadBackwards(ctx, ref)
}

// Close forwards to the inner store when it is closable.
func (f *FaultInjector) Close() error {
	if err := f.arm(MethodClose); err != nil {
		return err
	}

	closer, ok := f.inner.(interface{ Close() error })
	if !ok {
		return nil
	}

	return closer.Close()
}
