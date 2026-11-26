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
		e.applyRemoteLWW(op, lwwKey(op.Collection, op.Key), func() {
			if mb, ok := e.local.(metaengine.MapBackend); ok {
				_ = mb.MapSet(ctx, op.Collection, op.Key, op.Value)
			}
		})

	case OpMapDelete:
		e.applyRemoteLWW(op, lwwKey(op.Collection, op.Key), func() {
			if mb, ok := e.local.(metaengine.MapBackend); ok {
				_ = mb.MapDelete(ctx, op.Collection, op.Key)
			}
		})

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

// applyRemoteLWW applies an incoming op to one LWW register while holding
// e.mu across the guard, the apply and the record. The guard is only sound
// as one critical section: with separate sections two ops for the same key
// can both pass it, and the older one then applies last and wins on this
// node alone.
func (e *replicatedEngine) applyRemoteLWW(op WriteOp, key string, apply func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if op.Timestamp.Before(e.timestamps[key]) {
		return
	}

	apply()

	if op.Timestamp.After(e.timestamps[key]) {
		e.timestamps[key] = op.Timestamp
	}
}
