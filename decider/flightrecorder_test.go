package decider_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// deciderFRMu serializes flight recorder tests (process-global constraint).
var deciderFRMu sync.Mutex

// deciderSafeBuf is a mutex-protected bytes.Buffer for concurrent access.
type deciderSafeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *deciderSafeBuf) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return sb.buf.Write(p)
}

func (sb *deciderSafeBuf) Len() int {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return sb.buf.Len()
}

func TestRepository_FlightRecorder_CapturesOnError(t *testing.T) {
	deciderFRMu.Lock()
	defer deciderFRMu.Unlock()

	var buf deciderSafeBuf

	recorder, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	if err := recorder.Start(); err != nil {
		t.Fatalf("recorder Start: %v", err)
	}
	t.Cleanup(recorder.Stop)

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithFlightRecorder[counterState](recorder,
			flightrecorder.OnError()),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()
	execErr := repo.Execute(
		context.Background(), streamID, "Counter",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return nil, errors.New("command rejected")
		},
	)
	if execErr == nil {
		t.Fatal("expected Execute error")
	}

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected flight recorder snapshot after Execute error")
	}
}

func TestRepository_FlightRecorder_NoCaptureOnSuccess(t *testing.T) {
	deciderFRMu.Lock()
	defer deciderFRMu.Unlock()

	var buf deciderSafeBuf

	recorder, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	if err := recorder.Start(); err != nil {
		t.Fatalf("recorder Start: %v", err)
	}
	t.Cleanup(recorder.Stop)

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithFlightRecorder[counterState](recorder,
			flightrecorder.OnError()),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()
	_ = repo.Execute(
		context.Background(), streamID, "Counter",
		func(_ counterState, version event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", streamID, version+1)}, nil
		},
	)

	time.Sleep(200 * time.Millisecond)

	if buf.Len() != 0 {
		t.Fatal("expected NO flight recorder snapshot on successful Execute")
	}
}

func TestRepository_FlightRecorder_NilRecorder(t *testing.T) {
	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithFlightRecorder[counterState](nil, nil),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()
	err = repo.Execute(
		context.Background(), streamID, "Counter",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return nil, errors.New("should not crash")
		},
	)
	if err == nil {
		t.Fatal("expected Execute error")
	}
}
