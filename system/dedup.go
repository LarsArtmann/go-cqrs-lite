package system

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Dedup resolves the deployment's deduplication capability (ADR-0158): the
// engine named "dedup" when present, else the first configured engine —
// capability-asserted to metaengine.DedupStore (the ADR-0142 CAS window:
// DedupCheckAndRecord/DedupSeen/DedupSweep). It is the migration target for
// the deprecated idempotency/{kvstore,sqlstore} modules. Map-shaped engines
// resolve to MapDedupStore — degraded semantics, declared in
// metaengine.DegradedADTs, same TTL-window contract.
//
// The resolution mirrors checkpointEngine() (named-else-first with runtime
// assertion); the error names the engines inspected so a missing capability
// is a deployment-declaration fix, not a mystery.
func (s *System) Dedup() (metaengine.DedupStore, error) {
	var engine metaengine.Engine

	var inspected []string

	for _, ne := range s.engines {
		inspected = append(inspected, ne.name)

		if ne.name == "dedup" {
			engine = ne.engine

			break
		}
	}

	if engine == nil && len(s.engines) > 0 {
		engine = s.engines[0].engine
	}

	if engine == nil {
		return nil, fmt.Errorf(
			"system: no engines configured; Dedup needs a deployment engine with the DedupStore capability",
		)
	}

	dedup, ok := engine.(metaengine.DedupStore)
	if !ok {
		name := ""
		for _, ne := range s.engines {
			if ne.engine == engine {
				name = ne.name

				break
			}
		}

		return nil, fmt.Errorf(
			"system: engine %q does not implement metaengine.DedupStore (inspected: %v) — name a dedup-capable engine %q in the deployment, or use a map engine's MapDedupStore",
			name,
			inspected,
			"dedup",
		)
	}

	return dedup, nil
}
