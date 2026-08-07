package system

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Drainer stops accepting new work and finishes in-flight work, bounded by ctx.
// Implemented by resources that need to drain before connections close (e.g.,
// event subscribers, projection runners). Drain is called by [System.GracefulClose]
// BEFORE [System.Close], so in-flight handlers complete before their underlying
// connections are dropped.
type Drainer interface {
	Drain(ctx context.Context) error
}

// shutdownEdge declares that Before must close before After during Close().
// Resource names are engine names from DeploymentConfig.Engines. The projection
// host always closes first (before any engine) and cannot participate in
// shutdown dependency edges.
type shutdownEdge struct {
	before, after string
}

// orderedEngines returns engines sorted by shutdown dependencies. Engines not
// in any dependency edge keep their creation order. Cycles fall back to
// creation order for the affected engines.
func (s *System) orderedEngines() []metaengine.Engine {
	if len(s.shutdownDeps) == 0 || len(s.engines) == 0 {
		return s.engineSlice()
	}

	// Build a set of unique engine indices by name.
	nameToIdx := make(map[string]int, len(s.engines))
	for i, ne := range s.engines {
		if _, exists := nameToIdx[ne.name]; !exists {
			nameToIdx[ne.name] = i
		}
	}

	// Build adjacency list from dependency edges.
	after := make([][]int, len(s.engines))
	inDegree := make([]int, len(s.engines))

	for _, edge := range s.shutdownDeps {
		beforeIdx, beforeOK := nameToIdx[edge.before]
		afterIdx, afterOK := nameToIdx[edge.after]
		if !beforeOK || !afterOK || beforeIdx == afterIdx {
			continue
		}

		after[beforeIdx] = append(after[beforeIdx], afterIdx)
		inDegree[afterIdx]++
	}

	// Kahn's algorithm for topological sort.
	queue := make([]int, 0, len(s.engines))
	for i := range s.engines {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	result := make([]metaengine.Engine, 0, len(s.engines))
	processed := 0

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		result = append(result, s.engines[idx].engine)
		processed++

		for _, next := range after[idx] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// If there's a cycle, append remaining engines in creation order.
	// Cycle nodes always have inDegree > 0 (they were never dequeued).
	if processed < len(s.engines) {
		for i := range s.engines {
			if inDegree[i] > 0 {
				result = append(result, s.engines[i].engine)
			}
		}
	}

	return result
}

// RegisterDrainer registers a [Drainer] that will be called by [System.GracefulClose]
// before [System.Close]. Use this to ensure in-flight work (e.g., event subscribers,
// HTTP handlers) completes before infrastructure connections are dropped.
//
// Drainers are registered at runtime (after [New]) because they typically need
// access to the System (bus, event store) which is not available before
// construction completes.
func (s *System) RegisterDrainer(d Drainer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.drainers = append(s.drainers, d)
}
