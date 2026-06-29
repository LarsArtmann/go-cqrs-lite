package projectionhost_test

import (
	"context"
	"log/slog"
	"sync"
)

// logDebug makes capturingSlogHandler capture all records (debug and above).
const logDebug = slog.LevelDebug

// capturingSlogHandler records every Handle call so tests can assert that an
// injected logger actually received lifecycle events.
type capturingSlogHandler struct {
	mu    sync.Mutex
	level slog.Leveler
	recs  []slog.Record
}

func (h *capturingSlogHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level.Level()
}

func (h *capturingSlogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recs = append(h.recs, r.Clone())

	return nil
}

func (h *capturingSlogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingSlogHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *capturingSlogHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.recs)
}

func newSlogLogger(h slog.Handler) *slog.Logger {
	return slog.New(h)
}
