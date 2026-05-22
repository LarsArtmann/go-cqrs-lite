package event

import "context"

type replayKeyType struct{}

var replayKey replayKeyType

// WithReplay marks the context as a replay context.
func WithReplay(ctx context.Context, replay bool) context.Context {
	return context.WithValue(ctx, replayKey, replay)
}
