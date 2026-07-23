package projectionhost_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// TestVersionedSeekableJournal_WithProjectionHost proves that
// schema.VersionedSeekableJournal works when passed to projectionhost.New —
// the real-world composition this feature was built for. Events stored at
// schema version 1 are upcasted to version 2 before reaching the projection.
func TestVersionedSeekableJournal_WithProjectionHost(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	// Store events at schema version 1 with payload "v1".
	aggID := id.NewAggregateID()

	for range 3 {
		evt, _ := event.NewEvent(
			"test.upcast", aggID, "Test", event.Version(1), []byte("v1"),
			event.WithSchemaVersion(1),
		)
		journal.append(evt)
	}

	// Wrap the journal with an upcaster that transforms v1→v2.
	versionedJournal, err := schema.NewVersionedSeekableJournal(journal, v1ToV2Upcaster{})
	if err != nil {
		t.Fatalf("NewVersionedSeekableJournal: %v", err)
	}

	// Create a projection that captures the upcasted payload.
	proj := &payloadCaptureProjection{name: "upcast-test"}

	host, err := projectionhost.New(versionedJournal, cpStore, projectionhost.WithBatchSize(10))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = host.Start(ctx)
	}()

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 3
	})
	cancel()
	host.Stop()

	// All 3 events should have been upcasted: payload "v2", schema version 2.
	proj.mu.Lock()
	defer proj.mu.Unlock()

	if len(proj.payloads) != 3 {
		t.Fatalf("expected 3 captured payloads, got %d", len(proj.payloads))
	}

	for i, p := range proj.payloads {
		if p != "v2" {
			t.Errorf("event %d: payload = %q, want %q (upcaster not applied)", i, p, "v2")
		}
	}

	for i, sv := range proj.schemaVersions {
		if sv != 2 {
			t.Errorf("event %d: schemaVersion = %d, want 2 (upcaster not applied)", i, sv)
		}
	}
}

// v1ToV2Upcaster transforms events of type "test.upcast" from schema version 1
// to version 2, changing the payload from "v1" to "v2".
type v1ToV2Upcaster struct{}

func (v1ToV2Upcaster) SourceType() event.Type             { return "test.upcast" }
func (v1ToV2Upcaster) SourceVersion() event.SchemaVersion { return 1 }
func (v1ToV2Upcaster) Upcast(evt event.Event) (event.Event, error) {
	return event.NewEvent(
		evt.Type(),
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		[]byte("v2"),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(2),
	)
}

// payloadCaptureProjection captures the payload and schema version of every
// event it handles, so the test can verify the upcaster was applied.
type payloadCaptureProjection struct {
	name           string
	count          atomic.Int64
	payloads       []string
	schemaVersions []event.SchemaVersion
	mu             sync.Mutex
}

func (p *payloadCaptureProjection) Name() string             { return p.name }
func (p *payloadCaptureProjection) EventTypes() []event.Type { return nil }

func (p *payloadCaptureProjection) Handle(_ context.Context, evt event.Event) error {
	p.count.Add(1)
	p.mu.Lock()
	p.payloads = append(p.payloads, string(evt.Payload()))
	p.schemaVersions = append(p.schemaVersions, evt.SchemaVersion())
	p.mu.Unlock()

	return nil
}
