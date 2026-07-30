package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// C018: memory.NewMemoryStore() as journal fallback in type assertion fires.
func TestC018_SilentJournalFallback_TypeAssert(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func journalFromStore(store Store) Journal {
	j, ok := store.(Journal)
	if !ok {
		return memory.NewMemoryStore()
	}
	return j
}
`,
	})
	findings := runDetector(t, correctness.NewC018Detector(ctx))
	assertRule(t, findings, "C018", 1)
}

// C018: memory.NewMemoryStore() as journal fallback in type switch fires.
func TestC018_SilentJournalFallback_TypeSwitch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func journalFromStore(store Store) Journal {
	switch s := store.(type) {
	case Journal:
		return s
	default:
		return memory.NewMemoryStore()
	}
}
`,
	})
	findings := runDetector(t, correctness.NewC018Detector(ctx))
	assertRule(t, findings, "C018", 1)
}

// C018: No finding when NewMemoryStore is not in a journal fallback function.
func TestC018_NoFindingForNormalMemoryStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func newStore() Store {
	return memory.NewMemoryStore()
}
`,
	})
	findings := runDetector(t, correctness.NewC018Detector(ctx))
	assertRule(t, findings, "C018", 0)
}

// C018: SeekableJournal type assertion also triggers the detector.
func TestC018_SilentJournalFallback_SeekableJournal(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func journalFromStore(store Store) Journal {
	j, ok := store.(event.SeekableJournal)
	if !ok {
		return memory.NewMemoryStore()
	}
	return j
}
`,
	})
	findings := runDetector(t, correctness.NewC018Detector(ctx))
	assertRule(t, findings, "C018", 1)
}

// C021: DecodePayloadAuto while mutex is held fires.
func TestC021_MutexHeldDuringDecode(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, err := event.DecodePayloadAuto[Payload](evt)
	if err != nil {
		return err
	}
	r.data[p.ID] = p
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC021Detector(ctx))
	assertRule(t, findings, "C021", 1)
}

// C021: json.Unmarshal while mutex is held fires.
func TestC021_MutexHeldDuringJSONUnmarshal(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

import "encoding/json"

func (r *ReadModelStore) Handle(evt Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var p Payload
	err := json.Unmarshal(evt.Payload(), &p)
	if err != nil {
		return err
	}
	r.data[p.ID] = p
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC021Detector(ctx))
	assertRule(t, findings, "C021", 1)
}

// C021: No finding when decode is outside the lock.
func TestC021_NoFindingWhenDecodeOutsideLock(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	p, err := event.DecodePayloadAuto[Payload](evt)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[p.ID] = p
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC021Detector(ctx))
	assertRule(t, findings, "C021", 0)
}

// C021: No finding when no decode at all.
func TestC021_NoFindingWhenNoDecode(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.count++
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC021Detector(ctx))
	assertRule(t, findings, "C021", 0)
}

// C021: RLock held during decode also fires.
func TestC021_RLockHeldDuringDecode(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, err := event.DecodePayloadAuto[Payload](evt)
	if err != nil {
		return err
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC021Detector(ctx))
	assertRule(t, findings, "C021", 1)
}

// C024: Dual-write without transaction fires.
func TestC024_DualWriteWithoutTransaction(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	r.data[evt.ID()] = evt.Payload()
	r.syncToSQL()
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC024Detector(ctx))
	assertRule(t, findings, "C024", 1)
}

// C024: No finding when transaction is used.
func TestC024_NoFindingWithTransaction(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	r.data[evt.ID()] = evt.Payload()
	r.syncToSQL()
	tx.Commit()
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC024Detector(ctx))
	assertRule(t, findings, "C024", 0)
}

// C024: No finding when RunInTx is used as transaction wrapper.
func TestC024_NoFindingWithRunInTx(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	return RunInTx(db, func(tx Tx) error {
		r.data[evt.ID()] = evt.Payload()
		return r.syncToSQL()
	})
}
`,
	})
	findings := runDetector(t, correctness.NewC024Detector(ctx))
	assertRule(t, findings, "C024", 0)
}

// C024: No finding when no in-memory mutation.
func TestC024_NoFindingWithoutMutation(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

func (r *ReadModelStore) Handle(evt Event) error {
	r.syncToSQL()
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC024Detector(ctx))
	assertRule(t, findings, "C024", 0)
}

// C025: fmt.Errorf without %w in CQRS-importing file fires.
func TestC025_BareErrorfInCQRSFile(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func handle(evt event.Event) error {
	if evt == nil {
		return fmt.Errorf("event is nil")
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC025Detector(ctx))
	assertRule(t, findings, "C025", 1)
}

// C025: No finding when %w is used.
func TestC025_NoFindingWithWrap(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func handle(evt event.Event, err error) error {
	if err != nil {
		return fmt.Errorf("handle: %w", err)
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC025Detector(ctx))
	assertRule(t, findings, "C025", 0)
}

// C025: No finding when file doesn't import CQRS modules.
func TestC025_NoFindingInNonCQRSFile(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "fmt"

func handle(id string) error {
	if id == "" {
		return fmt.Errorf("empty id")
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC025Detector(ctx))
	assertRule(t, findings, "C025", 0)
}

// C026: TTL constant defined but different literal passed to NewMemoryStore.
func TestC026_TTLMismatch_NewMemoryStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

import (
	"idempotency"
	"time"
)

const idempotencyTTL = 5 * time.Minute

func setup() {
	store := idempotency.NewMemoryStore(24 * time.Hour)
	_ = store
}
`,
	})
	findings := runDetector(t, correctness.NewC026Detector(ctx))
	assertRule(t, findings, "C026", 1)
}

// C026: TTL constant used correctly — no finding.
func TestC026_NoFindingWhenConstUsed(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

import (
	"idempotency"
	"time"
)

const idempotencyTTL = 5 * time.Minute

func setup() {
	store := idempotency.NewMemoryStore(idempotencyTTL)
	_ = store
}
`,
	})
	findings := runDetector(t, correctness.NewC026Detector(ctx))
	assertRule(t, findings, "C026", 0)
}

// C026: TTL mismatch in middleware.CommandIdempotency.
func TestC026_TTLMismatch_Middleware(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

import (
	"middleware"
	"time"
)

const idempotencyTTL = 5 * time.Minute

func setup(d Dispatcher) {
	d.Use(middleware.CommandIdempotency(store, 24*time.Hour, nil))
}
`,
	})
	findings := runDetector(t, correctness.NewC026Detector(ctx))
	assertRule(t, findings, "C026", 1)
}

// C026: TTL mismatch in middleware.EventIdempotency.
func TestC026_TTLMismatch_EventIdempotency(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

import (
	"middleware"
	"time"
)

const idempotencyTTL = 5 * time.Minute

func setup(bus Bus) {
	bus.Use(middleware.EventIdempotency(store, 24*time.Hour, nil))
}
`,
	})
	findings := runDetector(t, correctness.NewC026Detector(ctx))
	assertRule(t, findings, "C026", 1)
}

// C026: TTL mismatch in middleware.QueryIdempotency.
func TestC026_TTLMismatch_QueryIdempotency(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

import (
	"middleware"
	"time"
)

const idempotencyTTL = 5 * time.Minute

func setup(q Q) {
	q.Use(middleware.QueryIdempotency(store, 24*time.Hour, keyFn))
}
`,
	})
	findings := runDetector(t, correctness.NewC026Detector(ctx))
	assertRule(t, findings, "C026", 1)
}

// C026: No TTL constant defined — no finding.
func TestC026_NoFindingWithoutTTLConst(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

import (
	"idempotency"
	"time"
)

func setup() {
	store := idempotency.NewMemoryStore(24 * time.Hour)
	_ = store
}
`,
	})
	findings := runDetector(t, correctness.NewC026Detector(ctx))
	assertRule(t, findings, "C026", 0)
}

// C027: bus.Subscribe alongside projectionhost.New fires.
func TestC027_BusSubscriptionAlongsideProjectionHost(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	host := projectionhost.New(journal, cpStore)
	host.Register(&MyProjection{})

	bus.Subscribe("user.created", handlerFunc)
}
`,
	})
	findings := runDetector(t, correctness.NewC027Detector(ctx))
	assertRule(t, findings, "C027", 1)
}

// C027: No finding when projectionhost is not used.
func TestC027_NoFindingWithoutProjectionHost(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.Subscribe("user.created", handlerFunc)
}
`,
	})
	findings := runDetector(t, correctness.NewC027Detector(ctx))
	assertRule(t, findings, "C027", 0)
}

// C027: No finding when only projectionhost is used (no bus.Subscribe).
func TestC027_NoFindingWithOnlyProjectionHost(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	host := projectionhost.New(journal, cpStore)
	host.Register(&MyProjection{})
}
`,
	})
	findings := runDetector(t, correctness.NewC027Detector(ctx))
	assertRule(t, findings, "C027", 0)
}
