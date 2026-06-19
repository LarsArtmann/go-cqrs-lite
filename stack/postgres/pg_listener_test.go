package postgres

import (
	"context"
	"errors"
	"testing"
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
