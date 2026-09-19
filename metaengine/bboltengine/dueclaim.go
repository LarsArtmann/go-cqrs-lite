package bboltengine

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// wireMapRuntimes attaches the shared ADR-0142 Map runtimes: the embedded
// *metaengine.MapDueClaimer and *metaengine.MapDedupStore promote their
// methods, so the engine satisfies metaengine.DueClaimer and
// metaengine.DedupStore with zero engine-specific claim code. Semantics are
// pinned by adttest.AssertDueClaimer / AssertDedupStore (dueclaim_test.go).
//art-dupl:accept each dep-isolated engine module wires the shared runtimes; per-module init is the database/sql registration pattern (AGENTS contract 19)
func (e *bboltEngine) wireMapRuntimes() error {
	claims, err := metaengine.NewMapDueClaimer(e)
	if err != nil {
		return fmt.Errorf("bboltengine: wire claims: %w", err)
	}

	dedup, err := metaengine.NewMapDedupStore(e)
	if err != nil {
		return fmt.Errorf("bboltengine: wire dedup: %w", err)
	}

	e.MapDueClaimer = claims
	e.MapDedupStore = dedup

	return nil
}
