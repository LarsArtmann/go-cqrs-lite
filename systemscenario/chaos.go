package systemscenario

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// delayedDriverSeq makes every DelayedDriver registration process-unique:
// the metaengine driver registry has no unregister hook, so per-call names
// (delayed-memory-1, delayed-memory-2, ...) keep repeated registrations in
// the same test binary from clashing.
var delayedDriverSeq atomic.Int64

// DelayedDriver wraps a registered base driver with artificial journal
// latency and returns the wrapper's driver name for EngineConfig{Driver:
// name} — the chaos seam for scenario tests. Every journal-family operation
// (StreamAppend, StreamRead, StreamVersion, JournalReadAll, JournalReadFrom,
// the atomic and transactional save paths, seq-seek reads, event-ID loads,
// temporal reads) sleeps for delay before delegating to the base engine;
// map operations pass through undelayed so read models stay fast (the chaos
// targets the journal I/O path — the plan's "DelayedJournal").
//
//	deploy := systemscenario.Memory()
//	deploy.Engines["primary"] = system.EngineConfig{
//		Driver: systemscenario.DelayedDriver(t, "memory", 2*time.Millisecond),
//	}
//
// Capability forwarding: Go does not tunnel type assertions through
// interface embedding, so the wrapper forwards, by hand, exactly the
// capabilities system wiring and the adapters discover via assertion —
// StreamLogBackend, AtomicAppender, Transactional, SeqSeekableStreamLog,
// EventByIDBackend, StreamTemporalReader, MapBackend, and MapUpdater.
// Booting a delayed deployment at all proves the atomicity gate
// (AtomicAppender or Transactional) survives the wrap; all other engine
// capabilities (vectors, search, spatial, snapshots, timers, calibration)
// are intentionally dropped, so DelayedDriver is for stream-log chaos
// scenarios, not full-surface engine tests. A base engine missing a
// forwarded capability fails loudly on first use, never silently.
//
// The registration is process-global and never removed (the registry has
// no unregister hook); the name embeds a sequence number so repeated calls
// in one test binary are safe. The base driver must already be registered
// (import its package first), which is checked eagerly.
func DelayedDriver(t testing.TB, base string, delay time.Duration) string {
	t.Helper()

	if delay < 0 {
		t.Fatalf("DelayedDriver: delay must be >= 0, got %s", delay)
	}

	if _, err := metaengine.LookupDriver(base); err != nil {
		t.Fatalf("DelayedDriver: base driver %q: %v", base, err)
	}

	name := fmt.Sprintf("systemscenario/delayed-%s-%d", base, delayedDriverSeq.Add(1))

	metaengine.RegisterDriver(name,
		func(ctx context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			factory, err := metaengine.LookupDriver(base)
			if err != nil {
				return nil, fmt.Errorf("systemscenario: delayed driver %q: %w", name, err)
			}

			inner, err := factory(ctx, cfg)
			if err != nil {
				return nil, err
			}

			return &delayedEngine{Engine: inner, delay: delay}, nil
		})

	return name
}

// delayedEngine injects latency into journal operations. It embeds the base
// [metaengine.Engine] (Profile, Close) and forwards the capability
// interfaces the system package discovers by type assertion — see
// [DelayedDriver] for why each must be forwarded explicitly.
type delayedEngine struct {
	metaengine.Engine

	delay time.Duration
}

// pause sleeps for the configured delay, honoring ctx cancellation so a
// canceled context does not stretch the latency into the await timeout.
func (d *delayedEngine) pause(ctx context.Context) {
	if d.delay <= 0 {
		return
	}

	timer := time.NewTimer(d.delay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

// streamBackend resolves the base's StreamLogBackend, failing loudly when
// the base lacks it (a delayed driver over a non-stream engine is a
// configuration mistake, not a degraded mode).
func (d *delayedEngine) streamBackend(op string) (metaengine.StreamLogBackend, error) {
	backend, ok := d.Engine.(metaengine.StreamLogBackend)
	if !ok {
		return nil, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks StreamLogBackend (needed for %s)",
			d.Engine, op,
		)
	}

	return backend, nil
}

func (d *delayedEngine) StreamAppend(
	ctx context.Context, collection, streamID string, values []any,
) error {
	d.pause(ctx)

	backend, err := d.streamBackend("StreamAppend")
	if err != nil {
		return err
	}

	return backend.StreamAppend(ctx, collection, streamID, values)
}

func (d *delayedEngine) StreamRead(
	ctx context.Context, collection, streamID string,
) ([]any, error) {
	d.pause(ctx)

	backend, err := d.streamBackend("StreamRead")
	if err != nil {
		return nil, err
	}

	return backend.StreamRead(ctx, collection, streamID)
}

func (d *delayedEngine) StreamVersion(
	ctx context.Context, collection, streamID string,
) (int64, error) {
	d.pause(ctx)

	backend, err := d.streamBackend("StreamVersion")
	if err != nil {
		return 0, err
	}

	return backend.StreamVersion(ctx, collection, streamID)
}

func (d *delayedEngine) JournalReadAll(ctx context.Context, collection string) ([]any, error) {
	d.pause(ctx)

	backend, err := d.streamBackend("JournalReadAll")
	if err != nil {
		return nil, err
	}

	return backend.JournalReadAll(ctx, collection)
}

func (d *delayedEngine) JournalReadFrom(
	ctx context.Context, collection string, afterSeq int64, limit int,
) ([]any, error) {
	d.pause(ctx)

	backend, err := d.streamBackend("JournalReadFrom")
	if err != nil {
		return nil, err
	}

	return backend.JournalReadFrom(ctx, collection, afterSeq, limit)
}

func (d *delayedEngine) StreamAppendExpected(
	ctx context.Context, collection, streamID string, expectedVersion int64, values []any,
) error {
	d.pause(ctx)

	appender, ok := d.Engine.(metaengine.AtomicAppender)
	if !ok {
		return fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks AtomicAppender (needed for StreamAppendExpected)",
			d.Engine,
		)
	}

	return appender.StreamAppendExpected(ctx, collection, streamID, expectedVersion, values)
}

func (d *delayedEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	d.pause(ctx)

	tx, ok := d.Engine.(metaengine.Transactional)
	if !ok {
		return fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks Transactional (needed for RunInTx)",
			d.Engine,
		)
	}

	return tx.RunInTx(ctx, fn)
}

func (d *delayedEngine) JournalReadAllWithSeq(
	ctx context.Context, collection string,
) ([]metaengine.StreamLogEntry, error) {
	d.pause(ctx)

	seeker, ok := d.Engine.(metaengine.SeqSeekableStreamLog)
	if !ok {
		return nil, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks SeqSeekableStreamLog (needed for JournalReadAllWithSeq)",
			d.Engine,
		)
	}

	return seeker.JournalReadAllWithSeq(ctx, collection)
}

func (d *delayedEngine) JournalReadFromSeq(
	ctx context.Context, collection string, afterSeq int64, limit int,
) ([]metaengine.StreamLogEntry, error) {
	d.pause(ctx)

	seeker, ok := d.Engine.(metaengine.SeqSeekableStreamLog)
	if !ok {
		return nil, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks SeqSeekableStreamLog (needed for JournalReadFromSeq)",
			d.Engine,
		)
	}

	return seeker.JournalReadFromSeq(ctx, collection, afterSeq, limit)
}

func (d *delayedEngine) StreamLoadByEventID(
	ctx context.Context, collection, eventID string,
) (any, error) {
	d.pause(ctx)

	loader, ok := d.Engine.(metaengine.EventByIDBackend)
	if !ok {
		return nil, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks EventByIDBackend (needed for StreamLoadByEventID)",
			d.Engine,
		)
	}

	return loader.StreamLoadByEventID(ctx, collection, eventID)
}

func (d *delayedEngine) StreamReadAsOfVersion(
	ctx context.Context, collection, streamID string, maxVersion int64,
) ([]any, error) {
	d.pause(ctx)

	reader, ok := d.Engine.(metaengine.StreamTemporalReader)
	if !ok {
		return nil, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks StreamTemporalReader (needed for StreamReadAsOfVersion)",
			d.Engine,
		)
	}

	return reader.StreamReadAsOfVersion(ctx, collection, streamID, maxVersion)
}

func (d *delayedEngine) StreamReadFromVersion(
	ctx context.Context, collection, streamID string, minVersion int64,
) ([]any, error) {
	d.pause(ctx)

	reader, ok := d.Engine.(metaengine.StreamTemporalReader)
	if !ok {
		return nil, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks StreamTemporalReader (needed for StreamReadFromVersion)",
			d.Engine,
		)
	}

	return reader.StreamReadFromVersion(ctx, collection, streamID, minVersion)
}

func (d *delayedEngine) MapSet(ctx context.Context, collection string, key, value any) error {
	backend, ok := d.Engine.(metaengine.MapBackend)
	if !ok {
		return fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks MapBackend (needed for MapSet)",
			d.Engine,
		)
	}

	return backend.MapSet(ctx, collection, key, value)
}

func (d *delayedEngine) MapGet(
	ctx context.Context, collection string, key any,
) (any, bool, error) {
	backend, ok := d.Engine.(metaengine.MapBackend)
	if !ok {
		return nil, false, fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks MapBackend (needed for MapGet)",
			d.Engine,
		)
	}

	return backend.MapGet(ctx, collection, key)
}

func (d *delayedEngine) MapDelete(ctx context.Context, collection string, key any) error {
	backend, ok := d.Engine.(metaengine.MapBackend)
	if !ok {
		return fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks MapBackend (needed for MapDelete)",
			d.Engine,
		)
	}

	return backend.MapDelete(ctx, collection, key)
}

func (d *delayedEngine) MapUpdate(
	ctx context.Context, collection string, key any, update func(prev any) any,
) error {
	updater, ok := d.Engine.(metaengine.MapUpdater)
	if !ok {
		return fmt.Errorf(
			"systemscenario: delayed driver: base engine %T lacks MapUpdater (needed for MapUpdate)",
			d.Engine,
		)
	}

	return updater.MapUpdate(ctx, collection, key, update)
}
