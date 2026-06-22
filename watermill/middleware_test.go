package watermill_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

func TestCorrelationIDMiddleware_NotNil(t *testing.T) {
	t.Parallel()

	mw := cqrswatermill.CorrelationIDMiddleware()
	if mw == nil {
		t.Fatal("CorrelationIDMiddleware returned nil")
	}
}

func TestNewRetryMiddleware_RetriesOnFailure(t *testing.T) {
	t.Parallel()

	cfg := cqrswatermill.RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	mw := cqrswatermill.NewRetryMiddleware(cfg)

	var callCount atomic.Int32
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		count := callCount.Add(1)

		if count < 3 { //nolint:mnd // fail first 2 calls
			return nil, errors.New("transient failure")
		}

		return []*message.Message{msg}, nil
	})

	msg := message.NewMessage("test-id", []byte("payload"))
	msg.SetContext(context.Background())
	produced, err := handler(msg)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if len(produced) != 1 {
		t.Fatalf("expected 1 message produced, got %d", len(produced))
	}

	if got := callCount.Load(); got != 3 { //nolint:mnd // 2 failures + 1 success
		t.Errorf("handler called %d times, want 3", got)
	}
}

func TestNewRetryMiddleware_ExhaustsRetries(t *testing.T) {
	t.Parallel()

	cfg := cqrswatermill.RetryConfig{
		MaxRetries:      2,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     5 * time.Millisecond,
		Multiplier:      2.0,
	}

	mw := cqrswatermill.NewRetryMiddleware(cfg)

	var callCount atomic.Int32
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		callCount.Add(1)

		return nil, errors.New("permanent failure")
	})

	msg := message.NewMessage("test-id", []byte("payload"))
	msg.SetContext(context.Background())
	_, err := handler(msg)

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	// MaxRetries=2 means 1 initial + 2 retries = 3 total calls
	if got := callCount.Load(); got != 3 { //nolint:mnd // 1 + 2 retries
		t.Errorf("handler called %d times, want 3", got)
	}
}

func TestDefaultRetryConfig_Values(t *testing.T) {
	t.Parallel()

	cfg := cqrswatermill.DefaultRetryConfig()

	if cfg.MaxRetries != 5 { //nolint:mnd // documented default
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}

	if cfg.InitialInterval != 100*time.Millisecond { //nolint:mnd // documented default
		t.Errorf("InitialInterval = %v, want 100ms", cfg.InitialInterval)
	}

	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", cfg.Multiplier)
	}
}
