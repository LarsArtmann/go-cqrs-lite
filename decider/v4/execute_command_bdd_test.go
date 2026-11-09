package decider_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

var errCmdRejected = errors.New("command rejected")

// auditedCommand is a consumer-style command satisfying decider.CausedCommand
// AND the optional Type capability structurally — command/ is deliberately
// NOT imported: the decider sees only the capabilities it needs.
type auditedCommand struct {
	id      id.CommandID
	cmdType record.Type
}

func (c *auditedCommand) ID() id.CommandID  { return c.id }
func (c *auditedCommand) Type() record.Type { return c.cmdType }

// bareCommand satisfies only the required ID capability — the stamp must
// degrade gracefully (ID without type).
type bareCommand struct {
	id id.CommandID
}

func (c *bareCommand) ID() id.CommandID { return c.id }

func newAuditedCommand(cmdType record.Type) *auditedCommand {
	return &auditedCommand{id: id.NewCommandID(), cmdType: cmdType}
}

func auditedCounterDecider() decider.Decider[int] {
	return decider.Decider[int]{
		Initial: 0,
		Apply: func(state int, evt event.Event) (int, error) {
			if evt.Type() == "CounterIncremented" {
				return state + 1, nil
			}

			return state, nil
		},
	}
}

var _ = Describe("ExecuteCommandRef", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
		bus   *eventtest.FakeBus
		repo  *decider.Repository[int]
		ref   id.StreamRef
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		bus = eventtest.NewFakeBus()

		var err error
		repo, err = decider.NewRepository(store, bus, auditedCounterDecider())
		Expect(err).ToNot(HaveOccurred())

		ref = id.NewStreamRef("Counter", id.NewStreamID())
	})

	persistedEvents := func() []event.Event {
		evts, readErr := store.ReadAll(ctx)
		Expect(readErr).ToNot(HaveOccurred())

		return evts
	}

	It("stamps typed causation and the compat keys onto every emitted event", func() {
		cmd := newAuditedCommand("IncrementCounter")

		err := decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, v event.Version, _ *auditedCommand) ([]event.Event, error) {
				return []event.Event{
					makeCounterEvent("CounterCreated", ref.ID, v+1),
					makeCounterEvent("CounterIncremented", ref.ID, v+2),
				}, nil
			})
		Expect(err).ToNot(HaveOccurred())

		evts := persistedEvents()
		Expect(evts).To(HaveLen(2))
		for _, evt := range evts {
			rec := event.AsRecord(evt)
			Expect(rec.MetaData.CausationID).To(Equal(cmd.ID().String()))
			Expect(rec.MetaData.Cause.Kind).To(Equal(record.CauseCommand))
			Expect(rec.MetaData.Cause.ID).To(Equal(cmd.ID().String()))

			md := evt.Metadata()
			Expect(md.Causation).ToNot(BeNil())
			Expect(md.Causation.CommandType).To(Equal("IncrementCounter"))
			Expect(md.Causation.CommandID).To(Equal(cmd.ID()))
			Expect(md.Custom[event.MetadataKeyCommandID]).To(Equal(cmd.ID().String()))
			Expect(md.Custom[event.MetadataKeyCommandType]).To(Equal("IncrementCounter"))
		}
	})

	It("stamps the ID when the command exposes no Type method", func() {
		cmd := &bareCommand{id: id.NewCommandID()}

		err := decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, v event.Version, _ *bareCommand) ([]event.Event, error) {
				return []event.Event{makeCounterEvent("CounterCreated", ref.ID, v+1)}, nil
			})
		Expect(err).ToNot(HaveOccurred())

		evts := persistedEvents()
		Expect(evts).To(HaveLen(1))

		md := evts[0].Metadata()
		Expect(md.Causation).ToNot(BeNil())
		Expect(md.Causation.CommandID).To(Equal(cmd.ID()))
		Expect(md.Causation.CommandType).To(BeEmpty())
		Expect(md.Custom).ToNot(HaveKey(event.MetadataKeyCommandType))
		Expect(md.Custom[event.MetadataKeyCommandID]).To(Equal(cmd.ID().String()))
	})

	It("respects causation the decide function already set", func() {
		cmd := newAuditedCommand("IncrementCounter")
		otherID := id.NewCommandID()

		err := decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, v event.Version, _ *auditedCommand) ([]event.Event, error) {
				kept, evtErr := event.NewEvent("CounterCreated", ref.ID, "Counter", v+1,
					[]byte(`{}`), event.WithCausation("Other", otherID))
				Expect(evtErr).ToNot(HaveOccurred())

				return []event.Event{
					kept,
					makeCounterEvent("CounterIncremented", ref.ID, v+2),
				}, nil
			})
		Expect(err).ToNot(HaveOccurred())

		evts := persistedEvents()
		Expect(evts).To(HaveLen(2))
		Expect(evts[0].Metadata().Causation.CommandID).To(Equal(otherID))
		Expect(evts[1].Metadata().Causation.CommandID).To(Equal(cmd.ID()))
	})

	It("keeps the ContextEnricher path working alongside the stamp", func() {
		correlationID := id.NewCorrelationID()

		enrichedRepo, err := decider.NewRepository(store, bus, auditedCounterDecider(),
			decider.WithEnricher[int](func(ctx context.Context) []event.Option {
				return []event.Option{event.WithCorrelationID(correlationID)}
			}))
		Expect(err).ToNot(HaveOccurred())

		cmd := newAuditedCommand("IncrementCounter")

		err = decider.ExecuteCommandRef(ctx, enrichedRepo, ref, cmd,
			func(_ int, v event.Version, _ *auditedCommand) ([]event.Event, error) {
				return []event.Event{makeCounterEvent("CounterCreated", ref.ID, v+1)}, nil
			})
		Expect(err).ToNot(HaveOccurred())

		evts := persistedEvents()
		Expect(evts).To(HaveLen(1))

		md := evts[0].Metadata()
		Expect(md.Tracing.CorrelationID).To(Equal(correlationID))
		Expect(md.Causation.CommandID).To(Equal(cmd.ID()))
	})

	It("persists nothing when decide returns zero events", func() {
		cmd := newAuditedCommand("NoopCommand")

		err := decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, _ event.Version, _ *auditedCommand) ([]event.Event, error) {
				return nil, nil
			})
		Expect(err).ToNot(HaveOccurred())
		Expect(persistedEvents()).To(BeEmpty())
		Expect(bus.Published).To(BeEmpty())
	})

	It("propagates decide errors and persists nothing", func() {
		cmd := newAuditedCommand("RejectedCommand")

		err := decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, _ event.Version, _ *auditedCommand) ([]event.Event, error) {
				return nil, errCmdRejected
			})
		Expect(err).To(MatchError(errCmdRejected))
		Expect(persistedEvents()).To(BeEmpty())
		Expect(bus.Published).To(BeEmpty())
	})

	It("executes a zero-ID command unstamped, identical to ExecuteRef", func() {
		cmd := &bareCommand{}

		err := decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, v event.Version, _ *bareCommand) ([]event.Event, error) {
				return []event.Event{makeCounterEvent("CounterCreated", ref.ID, v+1)}, nil
			})
		Expect(err).ToNot(HaveOccurred())

		evts := persistedEvents()
		Expect(evts).To(HaveLen(1))

		md := evts[0].Metadata()
		Expect(md.Causation).To(BeNil())
		Expect(md.Custom).ToNot(HaveKey(event.MetadataKeyCommandID))
	})
})
