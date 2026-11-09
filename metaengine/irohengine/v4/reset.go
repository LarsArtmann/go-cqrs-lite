package irohengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ResetEngine implements [metaengine.EngineResetter] for the replicated
// wrapper. Policy (see the forwarding-policy audit in engine_passthrough.go):
//
//   - A reset is NODE-LOCAL. The replication protocol has no reset WriteOp —
//     a drop-everything op is not CRDT-safe (it must lose to nothing), so it
//     cannot converge leaderlessly. A cluster-wide revert means resetting
//     every node while writes are quiesced (operator orchestration), the same
//     out-of-band contract as re-seeding any CRDT fleet.
//   - It delegates to the LOCAL engine's [metaengine.EngineResetter] and
//     fails loudly when the local engine cannot be reset — silently
//     pretending a replicated engine reverted when only its wrapper did
//     would be the worst lie available.
//   - The LWW timestamp map is deliberately KEPT, not cleared: the recorded
//     timestamps act as tombstone barriers so stale in-flight peer writes
//     (timestamps older than the reset) are rejected by isLWWNewer instead of
//     re-landing on the freshly replayed state. Replayed writes mint fresh
//     (newer) wall-clock timestamps and replicate normally.
func (e *replicatedEngine) ResetEngine(ctx context.Context) error {
	resetter, ok := e.local.(metaengine.EngineResetter)
	if !ok {
		return fmt.Errorf(
			"irohengine.ResetEngine: local engine %T does not implement metaengine.EngineResetter — "+
				"a replicated reset is only as revertible as its local half",
			e.local,
		)
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		return fmt.Errorf("irohengine.ResetEngine: reset local engine: %w", err)
	}

	return nil
}

// Compile-time assertion: the wrapper satisfies the reset capability
// (conditional on the local engine satisfying it at runtime).
var _ metaengine.EngineResetter = (*replicatedEngine)(nil)
