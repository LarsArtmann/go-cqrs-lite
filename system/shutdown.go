package system

import (
	"context"
	"fmt"
	"io"
	"slices"

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
	named := s.orderedNamedEngines()
	engines := make([]metaengine.Engine, len(named))
	for i, ne := range named {
		engines[i] = ne.engine
	}
	return engines
}

// orderedNamedEngines returns the topologically sorted engines with their
// config-key names. ShutdownOrder uses the names so callers can correlate
// the output with ShutdownDependency.Before/After values.
func (s *System) orderedNamedEngines() []namedEngine {
	if len(s.shutdownDeps) == 0 || len(s.engines) == 0 {
		return slices.Clone(s.engines)
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

	result := make([]namedEngine, 0, len(s.engines))
	processed := 0

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		result = append(result, s.engines[idx])
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
	// The seen set is a defensive guard: it prevents accidental duplicate
	// appends if the cycle-detection logic is refactored.
	if processed < len(s.engines) {
		seen := make(map[int]bool, len(result))
		for i := range s.engines {
			if inDegree[i] > 0 && !seen[i] {
				result = append(result, s.engines[i])
				seen[i] = true
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

// drainAll snapshots the drainer list under the read lock, then calls Drain
// on each one. Returns the first error without wrapping so callers can apply
// their own context.
func (s *System) drainAll(ctx context.Context) error {
	s.mu.RLock()
	drainers := make([]Drainer, len(s.drainers))
	copy(drainers, s.drainers)
	s.mu.RUnlock()

	for _, d := range drainers {
		if err := d.Drain(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Drain calls all registered [Drainer]s to stop accepting new work and
// finish in-flight work, bounded by ctx. Unlike [System.GracefulClose],
// Drain does NOT close resources — use this for rolling deploys where the
// process stays alive but should reject new requests.
func (s *System) Drain(ctx context.Context) error {
	if err := s.drainAll(ctx); err != nil {
		return fmt.Errorf("system: drain: %w", err)
	}

	return nil
}

// RegisterCloser registers an external resource (e.g., a custom connection
// pool, file handle) for lifecycle management. The closer will be called
// during [System.Close] AFTER all engines are closed, in registration order.
// Use this when you have infrastructure that is not a metaengine.Engine but
// still needs to be shut down with the System.
func (s *System) RegisterCloser(name string, closer io.Closer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closers = append(s.closers, namedCloser{closer: closer, name: name})
}
