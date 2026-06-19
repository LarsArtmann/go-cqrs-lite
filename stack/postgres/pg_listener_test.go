package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestValidateChannelName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		channel string
		wantErr bool
	}{
		{"default", "cqrs_events", false},
		{"snake_case", "my_app_events_v2", false},
		{"mixed case", "CqrsEvents", false},
		{"single char", "x", false},
		{"underscore first", "_private", false},
		{"empty", "", true},
		{"starts with digit", "1events", true},
		{"has space", "cqrs events", true},
		{"has hyphen", "cqrs-events", true},
		{"has quote", `cqrs"events`, true},
		{"has semicolon (injection attempt)", "cqrs; DROP TABLE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateChannelName(tt.channel)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateChannelName(%q): err=%v, wantErr=%v", tt.channel, err, tt.wantErr)
			}
		})
	}
}

func TestPgxListener_CloseBeforeListen(t *testing.T) {
	t.Parallel()

	l := &PgxListener{
		notifications: make(chan string, 1),
		done:          make(chan struct{}),
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close before Listen: %v", err)
	}

	// Second close is a no-op (closeOnce guard).
	if err := l.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	if !l.closed.Load() {
		t.Fatal("closed flag should be true after Close")
	}

	// Listen after Close must fail.
	if err := l.Listen(context.Background(), "x"); !errors.Is(err, ErrListenerClosed) {
		t.Fatalf("Listen after Close: err=%v, want ErrListenerClosed", err)
	}
}

func TestPgxListener_BadChannelName(t *testing.T) {
	t.Parallel()

	// Construct without calling Listen — the channel name check happens first.
	l := &PgxListener{
		notifications: make(chan string, 1),
		done:          make(chan struct{}),
		pool:          nil, // never acquired because validation runs first
	}

	err := l.Listen(context.Background(), "bad channel!")
	if err == nil {
		t.Fatal("expected error for invalid channel name")
	}
}

// TestPgxListener_CloseDoesNotDeadlock is the regression test for the critical
// deadlock that existed before the cancelFn fix. The old Close() called
// conn.Release() (which does NOT interrupt WaitForNotification) then waited on
// <-l.done — hanging forever because the receive loop was stuck on the network.
//
// This test simulates the post-Listen state (cancelFn set, receiveLoop running)
// and asserts Close() returns within 2s. If the cancel-first fix regresses, the
// goroutine never exits and this test FAILS on timeout.
func TestPgxListener_CloseDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	// Simulate the state after Listen() was called: a child context with cancel,
	// and a "receiveLoop" goroutine blocked on ctx (mimicking WaitForNotification).
	ctx, cancel := context.WithCancel(context.Background())

	l := &PgxListener{
		notifications: make(chan string, 1),
		done:          make(chan struct{}),
		cancelFn:      cancel,
	}

	// Simulate receiveLoop: block on ctx (like WaitForNotification), then close done.
	go func() {
		<-ctx.Done()
		close(l.done)
		close(l.notifications)
	}()

	// Close must return within 2s. Old code deadlocked indefinitely here.
	done := make(chan error, 1)

	go func() {
		done <- l.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close() deadlocked: did not return within 2s (cancelFn regression)")
	}
}

// TestValidateChannelName_Property runs property-based tests for the channel
// name validator. The table test covers 11 hand-picked cases; this adds
// exhaustive random-input coverage.
func TestValidateChannelName_Property(t *testing.T) {
	t.Parallel()

	// Property 1: strings matching the valid Postgres identifier regex
	// [A-Za-z_][A-Za-z0-9_]* must NEVER produce an error.
	t.Run("valid_identifiers_pass", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(t *rapid.T) {
			s := rapid.StringMatching(`[A-Za-z_][A-Za-z0-9_]*`).Draw(t, "channel")
			if err := validateChannelName(s); err != nil {
				t.Fatalf("validateChannelName(%q): unexpected error %v", s, err)
			}
		})
	})

	// Property 2: strings starting with a digit must ALWAYS error (Postgres
	// rejects identifiers beginning with a digit without double-quoting).
	t.Run("digit_first_fails", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(t *rapid.T) {
			digit := rapid.StringMatching(`[0-9]`).Draw(t, "digit")
			rest := rapid.StringMatching(`[A-Za-z0-9_]*`).Draw(t, "rest")
			s := digit + rest
			if err := validateChannelName(s); err == nil {
				t.Fatalf("validateChannelName(%q): expected error for digit-first", s)
			}
		})
	})

	// Property 3: no panic on arbitrary input (fuzz-like safety).
	t.Run("never_panics", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(t *rapid.T) {
			s := rapid.String().Draw(t, "input")
			_ = validateChannelName(s) // must not panic
		})
	})
}

func TestPgxListener_ReconnectConfig(t *testing.T) {
	t.Parallel()

	t.Run("default enables reconnect", func(t *testing.T) {
		t.Parallel()

		l := &PgxListener{reconnectCfg: defaultReconnectConfig()}
		if !l.reconnectCfg.enabled {
			t.Fatal("default config should enable reconnect")
		}

		if l.reconnectCfg.maxAttempts != 10 {
			t.Fatalf("default maxAttempts: got %d, want 10", l.reconnectCfg.maxAttempts)
		}

		if l.reconnectCfg.initialBackoff != 1*time.Second {
			t.Fatalf("default initialBackoff: got %v, want 1s", l.reconnectCfg.initialBackoff)
		}

		if l.reconnectCfg.maxBackoff != 30*time.Second {
			t.Fatalf("default maxBackoff: got %v, want 30s", l.reconnectCfg.maxBackoff)
		}
	})

	t.Run("WithoutReconnect disables", func(t *testing.T) {
		t.Parallel()

		l := &PgxListener{reconnectCfg: defaultReconnectConfig()}
		WithoutReconnect()(l)
		if l.reconnectCfg.enabled {
			t.Fatal("WithoutReconnect should disable reconnect")
		}
	})

	t.Run("WithReconnect(0) disables", func(t *testing.T) {
		t.Parallel()

		l := &PgxListener{reconnectCfg: defaultReconnectConfig()}
		WithReconnect(0)(l)
		if l.reconnectCfg.enabled {
			t.Fatal("WithReconnect(0) should disable reconnect")
		}
	})

	t.Run("WithReconnect sets maxAttempts", func(t *testing.T) {
		t.Parallel()

		l := &PgxListener{reconnectCfg: defaultReconnectConfig()}
		WithReconnect(5)(l)
		if l.reconnectCfg.maxAttempts != 5 {
			t.Fatalf("maxAttempts: got %d, want 5", l.reconnectCfg.maxAttempts)
		}
	})

	t.Run("WithReconnectBackoff sets schedule", func(t *testing.T) {
		t.Parallel()

		l := &PgxListener{reconnectCfg: defaultReconnectConfig()}
		WithReconnectBackoff(500*time.Millisecond, 10*time.Second)(l)
		if l.reconnectCfg.initialBackoff != 500*time.Millisecond {
			t.Fatalf("initialBackoff: got %v, want 500ms", l.reconnectCfg.initialBackoff)
		}

		if l.reconnectCfg.maxBackoff != 10*time.Second {
			t.Fatalf("maxBackoff: got %v, want 10s", l.reconnectCfg.maxBackoff)
		}
	})
}

func TestBackoffDuration(t *testing.T) {
	t.Parallel()

	l := &PgxListener{reconnectCfg: reconnectConfig{
		enabled:        true,
		maxAttempts:    10,
		initialBackoff: 1 * time.Second,
		maxBackoff:     30 * time.Second,
	}}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // capped
		{6, 30 * time.Second},
		{10, 30 * time.Second},
	}

	for _, tt := range tests {
		got := l.backoffDuration(tt.attempt)
		if got != tt.want {
			t.Fatalf("backoffDuration(%d): got %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestBackoffDuration_Property(t *testing.T) {
	t.Parallel()

	l := &PgxListener{reconnectCfg: reconnectConfig{
		enabled:        true,
		maxAttempts:    10,
		initialBackoff: 1 * time.Second,
		maxBackoff:     30 * time.Second,
	}}

	rapid.Check(t, func(t *rapid.T) {
		attempt := rapid.IntRange(0, 20).Draw(t, "attempt")
		delay := l.backoffDuration(attempt)

		// Property 1: delay is always within [initialBackoff, maxBackoff].
		if delay < l.reconnectCfg.initialBackoff {
			t.Fatalf(
				"delay %v < initial %v for attempt %d",
				delay,
				l.reconnectCfg.initialBackoff,
				attempt,
			)
		}

		if delay > l.reconnectCfg.maxBackoff {
			t.Fatalf("delay %v > max %v for attempt %d", delay, l.reconnectCfg.maxBackoff, attempt)
		}

		// Property 2: delay is monotonically non-decreasing.
		if attempt > 0 {
			prev := l.backoffDuration(attempt - 1)
			if delay < prev {
				t.Fatalf("delay %v < prev %v for attempt %d", delay, prev, attempt)
			}
		}
	})
}
