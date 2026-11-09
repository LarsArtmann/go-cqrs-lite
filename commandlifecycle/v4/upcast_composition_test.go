package commandlifecycle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	memorystore "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// failedPayloadV2 models a FUTURE evolution of commandlifecycle.FailedPayload:
// an ErrorCode field is added after old lifecycle streams were already
// persisted. The upcast composition test proves such an evolution stays a
// COMPOSITION concern (DecorateStore + UpcastSourceTransform around the raw
// store) rather than new framework code (plan D3, 2026-09-13).
type failedPayloadV2 struct {
	CommandType string    `json:"commandType"`
	Error       string    `json:"error"`
	Attempt     int       `json:"attempt"`
	FailedAt    time.Time `json:"failedAt"`
	ErrorCode   string    `json:"errorCode"`
}

// writeLegacyFailedEvent simulates a stream written by an OLDER library
// version: command.failed at schema v1, no ErrorCode field, typed causation.
func writeLegacyFailedEvent(
	ctx context.Context,
	t *testing.T,
	g Gomega,
	store event.Store,
	ref id.StreamRef,
) event.Event {
	t.Helper()

	evt, err := event.New(
		commandlifecycle.TypeFailed,
		ref.ID,
		commandlifecycle.StreamTypeCommandLifecycle,
		event.Version(1),
		commandlifecycle.FailedPayload{
			CommandType: "create_user",
			Error:       "boom",
			Attempt:     1,
			FailedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		event.WithCausation("create_user", id.NewCommandID()),
	)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(store.Save(ctx, ref, []event.Event{evt}, 0)).To(Succeed())

	return evt
}

// upcastFailedV1toV2 is the schema-evolution upcaster: v1 payloads gain the
// ErrorCode field and move to schema v2. Identity, timestamp, and metadata
// (including the typed causation) are preserved.
func upcastFailedV1toV2(evt event.Event) (event.Event, error) {
	old, err := event.DecodePayloadAuto[commandlifecycle.FailedPayload](evt)
	if err != nil {
		return nil, err
	}

	evolved := failedPayloadV2{
		CommandType: old.CommandType,
		Error:       old.Error,
		Attempt:     old.Attempt,
		FailedAt:    old.FailedAt,
		ErrorCode:   "legacy_unknown",
	}

	return event.New(
		evt.Type(),
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		evolved,
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithMetadata(evt.Metadata()),
		event.WithSchemaVersion(2),
	)
}

func TestUpcastComposition_LoadSeesNewShape(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	ctx := context.Background()
	raw := memorystore.NewMemoryStore()
	ref := id.NewStreamRef(commandlifecycle.StreamTypeCommandLifecycle, id.NewStreamID())

	legacy := writeLegacyFailedEvent(ctx, t, g, raw, ref)

	upcasted := event.DecorateStore(raw, nil, schema.UpcastSourceTransform(
		schema.NewUpcaster(commandlifecycle.TypeFailed, 1, upcastFailedV1toV2),
	))

	evts, err := upcasted.Load(ctx, ref)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(evts).To(HaveLen(1))

	// The read side sees the NEW payload shape — old bytes, evolved view.
	decoded, err := event.DecodePayloadAuto[failedPayloadV2](evts[0])
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded.ErrorCode).To(Equal("legacy_unknown"))
	g.Expect(decoded.Error).To(Equal("boom"))
	g.Expect(evts[0].SchemaVersion()).To(Equal(event.SchemaVersion(2)))

	// Structural fidelity survives the upcast: identity, version, causation.
	g.Expect(evts[0].ID()).To(Equal(legacy.ID()))
	g.Expect(evts[0].Version()).To(Equal(legacy.Version()))
	g.Expect(evts[0].Metadata().Causation).ToNot(BeNil())
	g.Expect(evts[0].Metadata().Causation.CommandType).To(Equal("create_user"))

	// The raw store is untouched — evolution is a read-path view.
	rawEvts, err := raw.Load(ctx, ref)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rawEvts[0].SchemaVersion()).To(Equal(event.SchemaVersion(1)))
}

func TestUpcastComposition_ZeroUpcastersPassthrough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	ctx := context.Background()
	raw := memorystore.NewMemoryStore()
	ref := id.NewStreamRef(commandlifecycle.StreamTypeCommandLifecycle, id.NewStreamID())

	legacy := writeLegacyFailedEvent(ctx, t, g, raw, ref)

	passthrough := event.DecorateStore(raw, nil, schema.UpcastSourceTransform())

	evts, err := passthrough.Load(ctx, ref)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(evts).To(HaveLen(1))
	g.Expect(evts[0].Payload()).To(Equal(legacy.Payload()))
	g.Expect(evts[0].SchemaVersion()).To(Equal(event.SchemaVersion(1)))
}

func TestUpcastComposition_RecorderComposesOverUpcastStore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	ctx := context.Background()
	raw := memorystore.NewMemoryStore()

	cmd, err := command.New("create_user", id.NewStreamID())
	g.Expect(err).ToNot(HaveOccurred())

	// The Recorder appends to the command's OWN lifecycle stream
	// (CommandLifecycle/<cmd-id>) — simulate the legacy write there.
	ref := commandlifecycle.LifecycleStreamRef(cmd)

	writeLegacyFailedEvent(ctx, t, g, raw, ref)

	upcasted := event.DecorateStore(raw, nil, schema.UpcastSourceTransform(
		schema.NewUpcaster(commandlifecycle.TypeFailed, 1, upcastFailedV1toV2),
	))

	// A Recorder over the decorated store seeds its version counter through
	// the upcasted read path and appends at the correct next version.
	recorder := commandlifecycle.NewRecorder(upcasted, commandlifecycle.WithStrict())

	g.Expect(recorder.RecordFailed(ctx, cmd, errors.New("again"), 2)).To(Succeed())

	all, err := raw.Load(ctx, ref)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(all).To(HaveLen(2))
	g.Expect(all[1].Version()).To(Equal(event.Version(2)))

	// The write path is passthrough: the Recorder's own event is stored
	// byte-for-byte as it wrote it (schema v1 shape, no upcast on write).
	decoded, err := event.DecodePayloadAuto[commandlifecycle.FailedPayload](all[1])
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded.Error).To(Equal("again"))
	g.Expect(decoded.Attempt).To(Equal(2))
	g.Expect(all[1].SchemaVersion()).To(Equal(event.SchemaVersion(1)))
}
