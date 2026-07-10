package stack

import "io"

// WithShutdownDependency declares that `before` must be closed before `after`
// during Bundle.Close(). Both must be registered via other With* options
// (WithEventStore, WithCloser, etc.). Use this to express ordering constraints
// that the registration order cannot capture, e.g. "close the projection host
// before closing the event store it reads from."
//
// Resources not declared in any dependency edge close in registration order.
// If there is a cycle in the dependency graph, Close falls back to
// registration order for the affected resources.
func WithShutdownDependency(before, after io.Closer) Option {
	return func(b *Bundle) {
		b.shutdownDeps = append(b.shutdownDeps, shutdownEdge{
			before: before,
			after:  after,
		})
	}
}

// orderedClosers returns the closers sorted by shutdown dependencies.
// Closers not in any dependency edge keep their registration order.
// Cycles fall back to registration order for the affected nodes.
func (b *Bundle) orderedClosers() []io.Closer {
	if len(b.shutdownDeps) == 0 {
		return b.closers
	}

	// Build a set of unique closers (pointer identity).
	index := make(map[io.Closer]int)
	order := make([]io.Closer, 0, len(b.closers))

	for _, c := range b.closers {
		if c == nil {
			continue
		}

		if _, exists := index[c]; !exists {
			index[c] = len(order)
			order = append(order, c)
		}
	}

	// Build adjacency list from dependency edges.
	// Edge: before → after means "before closes first".
	after := make([][]int, len(order)) // after[i] = nodes that depend on i
	inDegree := make([]int, len(order))

	for _, edge := range b.shutdownDeps {
		beforeIdx, beforeOK := index[edge.before]

		afterIdx, afterOK := index[edge.after]
		if !beforeOK || !afterOK || beforeIdx == afterIdx {
			continue
		}

		after[beforeIdx] = append(after[beforeIdx], afterIdx)
		inDegree[afterIdx]++
	}

	// Kahn's algorithm for topological sort.
	// Start with all nodes that have no incoming edges, in registration order.
	queue := make([]int, 0, len(order))

	for i := range order {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	result := make([]io.Closer, 0, len(order))
	processed := 0

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		result = append(result, order[idx])
		processed++

		for _, next := range after[idx] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// If there's a cycle, append remaining nodes in registration order.
	if processed < len(order) {
		appended := make(map[int]bool)
		for _, c := range result {
			appended[index[c]] = true
		}

		for i, c := range order {
			if !appended[i] {
				result = append(result, c)
			}
		}
	}

	return result
}
