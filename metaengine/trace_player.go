package metaengine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// TraceSummary aggregates a trace's operation counts — the input shape for
// benchmark calibration (mix ratios, event/query cardinality).
type TraceSummary struct {
	Applies int
	Queries int
	ByName  map[string]int
}

// TraceStats summarizes recorded ops by kind and name.
func TraceStats(ops []TraceOp) TraceSummary {
	s := TraceSummary{ByName: make(map[string]int)}

	for _, op := range ops {
		switch op.Op {
		case TraceOpApply:
			s.Applies++
			s.ByName[op.Name]++
		case TraceOpQuery:
			s.Queries++
			s.ByName[op.Name]++
		}
	}

	return s
}

// ReadTrace parses JSON-lines trace records. Unknown op kinds are preserved
// (forward compatibility); ReplayTrace skips them.
func ReadTrace(r io.Reader) ([]TraceOp, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var ops []TraceOp

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var op TraceOp
		if err := json.Unmarshal(line, &op); err != nil {
			return nil, fmt.Errorf("metaengine.ReadTrace: line %d: %w", len(ops)+1, err)
		}

		ops = append(ops, op)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("metaengine.ReadTrace: %w", err)
	}

	return ops, nil
}

// TraceSink receives replayed trace operations.
type TraceSink interface {
	// Apply applies one event of the given type. The sink synthesizes the
	// payload (the trace deliberately does not serialize payloads).
	Apply(ctx context.Context, eventType string) error

	// Query executes one query by name. The sink synthesizes the input.
	Query(ctx context.Context, name string) error
}

// ReplayTrace replays ops into the sink sequentially, preserving order.
// Unknown op kinds are skipped. The first sink error aborts the replay.
func ReplayTrace(ctx context.Context, ops []TraceOp, sink TraceSink) error {
	for i, op := range ops {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("metaengine.ReplayTrace: %w", err)
		}

		switch op.Op {
		case TraceOpApply:
			if err := sink.Apply(ctx, op.Name); err != nil {
				return fmt.Errorf("metaengine.ReplayTrace: op %d apply %s: %w", i, op.Name, err)
			}
		case TraceOpQuery:
			if err := sink.Query(ctx, op.Name); err != nil {
				return fmt.Errorf("metaengine.ReplayTrace: op %d query %s: %w", i, op.Name, err)
			}
		}
	}

	return nil
}

// StoreTraceSink replays a trace against a Store. Payloads and query inputs
// are synthesized by the caller-supplied factories; seq is the 0-based
// occurrence index of that event type / query name within the replay.
func StoreTraceSink(
	store *Store,
	payloadFor func(eventType string, seq int) any,
	inputFor func(name string, seq int) any,
) TraceSink {
	return &storeSink{store: store, payloadFor: payloadFor, inputFor: inputFor, seq: make(map[string]int)}
}

type storeSink struct {
	store      *Store
	payloadFor func(string, int) any
	inputFor   func(string, int) any
	seq        map[string]int
}

func (s *storeSink) Apply(ctx context.Context, eventType string) error {
	n := s.seq[eventType]
	s.seq[eventType] = n + 1

	return s.store.Apply(ctx, eventType, s.payloadFor(eventType, n)) //nolint:wrapcheck
}

func (s *storeSink) Query(ctx context.Context, name string) error {
	n := s.seq[name]
	s.seq[name] = n + 1

	_, err := s.store.ExecuteCtx(ctx, s.inputFor(name, n))

	return err //nolint:wrapcheck
}
