package event

import (
	"errors"
	"testing"
	"time"
)

func TestNewOutboxPublisher_NilOutbox(t *testing.T) {
	t.Parallel()

	_, err := NewOutboxPublisher(nil, &stubPublisher{})
	if err == nil {
		t.Fatal("expected error for nil outbox")
	}

	if !errors.Is(err, ErrNilOutbox) {
		t.Errorf("error = %v, want ErrNilOutbox", err)
	}
}

func TestNewOutboxPublisher_NilBus(t *testing.T) {
	t.Parallel()

	_, err := NewOutboxPublisher(&stubOutbox{}, nil)
	if err == nil {
		t.Fatal("expected error for nil bus")
	}

	if !errors.Is(err, ErrNilBus) {
		t.Errorf("error = %v, want ErrNilBus", err)
	}
}

func TestNewOutboxPublisher_Defaults(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{})
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	if p.interval != time.Second {
		t.Fatalf("interval = %v, want 1s", p.interval)
	}

	if p.batchSize != 100 {
		t.Fatalf("batchSize = %d, want 100", p.batchSize)
	}
}

func TestNewOutboxPublisher_WithPollInterval(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(
		&stubOutbox{},
		&stubPublisher{},
		WithPollInterval(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	if p.interval != 50*time.Millisecond {
		t.Fatalf("interval = %v, want 50ms", p.interval)
	}
}

func TestNewOutboxPublisher_WithBatchSize(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{}, WithBatchSize(10))
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	if p.batchSize != 10 {
		t.Fatalf("batchSize = %d, want 10", p.batchSize)
	}
}

func TestNewOutboxPublisher_ZeroIntervalResetsToDefault(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{}, WithPollInterval(0))
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	if p.interval != time.Second {
		t.Fatalf("interval = %v, want 1s (default)", p.interval)
	}
}

func TestNewOutboxPublisher_ZeroBatchSizeResetsToDefault(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{}, WithBatchSize(0))
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	if p.batchSize != 100 {
		t.Fatalf("batchSize = %d, want 100 (default)", p.batchSize)
	}
}

func TestOutboxPublisher_StartStop(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(
		&stubOutbox{},
		&stubPublisher{},
		WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	err = p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	err = p.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOutboxPublisher_DoubleStart(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{})
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	err = p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = p.Close() }()

	err = p.Start()
	if err == nil {
		t.Fatal("expected error on double start")
	}

	if !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("error = %q, want ErrAlreadyStarted", err.Error())
	}
}

func TestOutboxPublisher_CloseWithoutStart(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{})
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	err = p.Close()
	if err != nil {
		t.Fatalf("Close without start: %v", err)
	}
}

func TestOutboxPublisher_DoubleClose(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{})
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	_ = p.Start()
	_ = p.Close()
	_ = p.Close()
}

func TestOutboxPublisher_StartAfterClose(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{})
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	_ = p.Start()
	_ = p.Close()

	err = p.Start()
	if err == nil {
		t.Fatal("expected error when starting after close")
	}

	if !errors.Is(err, ErrPublisherClosed) {
		t.Fatalf("error = %q, want ErrPublisherClosed", err.Error())
	}
}

func TestOutboxPublisher_StartAfterCloseWithoutStart(t *testing.T) {
	t.Parallel()

	p, err := NewOutboxPublisher(&stubOutbox{}, &stubPublisher{})
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	_ = p.Close()

	err = p.Start()
	if err == nil {
		t.Fatal("expected error when starting after close")
	}

	if !errors.Is(err, ErrPublisherClosed) {
		t.Fatalf("error = %q, want ErrPublisherClosed", err.Error())
	}
}
