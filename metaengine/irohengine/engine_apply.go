package irohengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// applyRemote dispatches an incoming WriteOp to the local engine.
// Calls the LOCAL engine directly (not the wrapper methods), so there is
// no re-entrancy risk of triggering another publish.
func (e *replicatedEngine) applyRemote(op WriteOp) {
	ctx := context.Background()

	switch op.Kind {
	case OpMapSet:
		if !e.isLWWNewer(op.Collection, op.Key, op.Timestamp) {
			return
		}
		if mb, ok := e.local.(metaengine.MapBackend); ok {
			_ = mb.MapSet(ctx, op.Collection, op.Key, op.Value)
		}
		e.recordLWW(op.Collection, op.Key, op.Timestamp)

	case OpMapDelete:
		if !e.isLWWNewer(op.Collection, op.Key, op.Timestamp) {
			return
		}
		if mb, ok := e.local.(metaengine.MapBackend); ok {
			_ = mb.MapDelete(ctx, op.Collection, op.Key)
		}
		e.recordLWW(op.Collection, op.Key, op.Timestamp)

	case OpSetAdd:
		if sb, ok := e.local.(metaengine.SetBackend); ok {
			_ = sb.SetAdd(ctx, op.Collection, op.Key)
		}

	case OpCounterInc:
		if cb, ok := e.local.(metaengine.CounterBackend); ok {
			_ = cb.CounterIncrement(ctx, op.Collection, op.Delta)
		}

	case OpMultiAdd:
		if mb, ok := e.local.(metaengine.MultimapBackend); ok {
			_ = mb.MultiAdd(ctx, op.Collection, op.Key, op.Value)
		}

	case OpLogAppend:
		if lb, ok := e.local.(metaengine.LogBackend); ok {
			_ = lb.LogAppend(ctx, op.Collection, op.Value)
		}

	case OpGraphAddEdge:
		e.applyRemoteGraphAdd(op)

	case OpGraphRemoveEdge:
		e.applyRemoteGraphRemove(op)
	}
}
