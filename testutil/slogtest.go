package testutil

import (
	"context"
	"log/slog"
	"sync"
)

// CapturingSlogHandler is a [slog.Handler] that records every Handle call so
// tests can assert that an injected logger actually received lifecycle events.
//
// It is safe for concurrent use. The zero value is NOT usable — construct with
// [NewCapturingSlogHandler].
//
// Example:
//
//	h := testutil.NewCapturingSlogHandler(slog.LevelDebug)
//	logger := slog.New(h)
//	projectionhost.WithLogger(logger)
//	// ... exercise the host ...
//	if h.Count() == 0 { t.Fatal("expected at least one log record") }
type CapturingSlogHandler struct {
	mu    sync.Mutex
	level slog.Leveler
	recs  []slog.Record
}

// NewCapturingSlogHandler creates a capturing handler that enables records at
// or above level. Pass slog.LevelDebug to capture everything.
func NewCapturingSlogHandler(level slog.Leveler) *CapturingSlogHandler {
	return &CapturingSlogHandler{ //nolint:exhaustruct // recs is nil-capable
		level: level,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *CapturingSlogHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level.Level()
}

// Handle appends a clone of the record to the capture buffer.
func (h *CapturingSlogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.recs = append(h.recs, r.Clone())

	return nil
}

// WithAttrs returns the receiver unchanged — capturing all attrs inline.
func (h *CapturingSlogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }

// WithGroup returns the receiver unchanged — groups are flattened into records.
func (h *CapturingSlogHandler) WithGroup(_ string) slog.Handler { return h }

// Count returns the number of captured records.
func (h *CapturingSlogHandler) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.recs)
}

// Records returns a copy of the captured records.
func (h *CapturingSlogHandler) Records() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make([]slog.Record, len(h.recs))
	copy(out, h.recs)

	return out
}
