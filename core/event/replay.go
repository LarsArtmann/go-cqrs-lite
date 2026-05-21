package event

import "context"

type replayKeyType struct{}

var replayKey replayKeyType

// WithReplay marks the context as a replay context.
// Projection handlers can check IsReplay to distinguish
// replay (historical) events from live events.
func WithReplay(ctx context.Context, replay bool) context.Context {
	return context.WithValue(ctx, replayKey, replay)
}

// IsReplay returns true if the context was marked as a replay by WithReplay.
// Projection handlers use this to skip side effects during replay (e.g., sending emails).
func IsReplay(ctx context.Context) bool {
	val, ok := ctx.Value(replayKey).(bool)
	return ok && val
}
