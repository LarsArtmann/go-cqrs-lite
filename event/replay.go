package event

import "context"

type replayKeyType struct{}

var replayKey replayKeyType //nolint:gochecknoglobals // context key, standard Go pattern

// WithReplay marks the context as a replay context.
func WithReplay(ctx context.Context, replay bool) context.Context {
	return context.WithValue(ctx, replayKey, replay)
}

// IsReplay reports whether the context is marked as a replay context.
func IsReplay(ctx context.Context) bool {
	val, _ := ctx.Value(replayKey).(bool)
	return val
}
