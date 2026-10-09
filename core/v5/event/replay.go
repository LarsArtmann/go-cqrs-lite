package event

import "context"

type replayKeyType struct{}

var replayKey replayKeyType //nolint:gochecknoglobals // context key, standard Go pattern

// ProcessingMode indicates whether an event handler is processing live events
// or replayed historical events.
type ProcessingMode string

const (
	// ModeLive indicates normal (non-replay) event processing.
	ModeLive ProcessingMode = "live"
	// ModeReplay indicates replay of historical events (e.g., during projection rebuild).
	ModeReplay ProcessingMode = "replay"
)

// WithProcessingMode annotates the context with the specified processing mode.
// Handlers can query it via [IsReplay] or [ProcessingModeFrom].
func WithProcessingMode(ctx context.Context, mode ProcessingMode) context.Context {
	return context.WithValue(ctx, replayKey, mode == ModeReplay)
}

// ProcessingModeFrom returns the processing mode stored in the context.
// Returns [ModeLive] if no mode has been set.
func ProcessingModeFrom(ctx context.Context) ProcessingMode {
	if IsReplay(ctx) {
		return ModeReplay
	}

	return ModeLive
}

// IsReplay reports whether the context is marked as a replay context.
func IsReplay(ctx context.Context) bool {
	val, _ := ctx.Value(replayKey).(bool)
	return val
}
