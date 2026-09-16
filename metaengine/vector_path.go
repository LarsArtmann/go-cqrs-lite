package metaengine

import (
	"context"
	"fmt"
	"strings"
)

// Vector execution-path labels reported by [VectorPathReporter]. Both are
// O(N) brute-force classes — the label describes WHERE scoring happens, not
// an ANN index (native ANN paths are future work; ADR-0140).
const (
	// VectorPathPushdown: k-NN scoring pushed into the engine's SQL
	// (libSQL vector_distance_*, DuckDB array_* distance functions).
	VectorPathPushdown = "sql-pushdown"

	// VectorPathScan: rows scanned and scored in Go via VectorDistance
	// (modernc sqlite, MySQL, Postgres, Dgraph, KV/LSM engines, memory).
	VectorPathScan = "go-scan"
)

// VectorPathReporter is an optional capability for engines whose vector
// execution path is decided at runtime (driver probe, deployment config)
// rather than statically from the type. ExplainPlan and Doctor render the
// reported path so operators can see whether k-NN scores engine-side or
// in Go. Engines with a single fixed path may implement it too, so every
// vector-capable engine reports uniformly.
//
// The label describes the UNFILTERED VectorSearch path; metadata-filtered
// searches always evaluate filters in Go and may scan even on pushdown
// engines (the label does not change per query). Implementations return one
// of the two constants, or "" when the engine cannot report a path for its
// current wiring (e.g. a forwarding wrapper over a local engine that does
// not implement this interface) — rendering skips engines that report "".
type VectorPathReporter interface {
	// VectorSearchPath returns one of VectorPathPushdown or VectorPathScan.
	VectorSearchPath() string
}

// vectorSearchPathLabel returns the reported path label for engines that
// implement VectorPathReporter, or "" when the engine cannot say.
func vectorSearchPathLabel(eng Engine) string {
	if vp, ok := eng.(VectorPathReporter); ok {
		return vp.VectorSearchPath()
	}

	return ""
}

// memoryEngine.VectorSearchPath lives here instead of memory_engine.go
// because memory_engine.go sits on the file-size ratchet baseline (growth
// forbidden); methods may be defined in any file of the package.

// VectorSearchPath reports the memory engine's brute-force in-process scan.
func (e *memoryEngine) VectorSearchPath() string {
	return VectorPathScan
}

// explainVectorSection renders the ExplainPlan vector block: one k-NN path
// line per engine reporting VectorSearchPath, plus the full-scan WARN for
// engines serving vectors without VectorCounter. Empty when no engine
// serves vectors at all (stores without the Vector ADT keep the old output).
func explainVectorSection(engines []Engine) string {
	var b strings.Builder

	var warnings []string

	anyVector := false

	for _, eng := range engines {
		if _, isVB := eng.(VectorBackend); !isVB {
			continue
		}

		anyVector = true

		if path := vectorSearchPathLabel(eng); path != "" {
			fmt.Fprintf(&b, "  %s: k-NN path %s\n", eng.Profile().Name, path)
		}

		if _, counts := eng.(VectorCounter); !counts {
			warnings = append(warnings, fmt.Sprintf(
				"  WARN vector: %s serves k-NN by full scan and cannot report "+
					"collection sizes (no VectorCounter); latency grows linearly "+
					"with collection size\n", eng.Profile().Name))
		}
	}

	if !anyVector {
		return ""
	}

	return "\n--- Vectors ---\n" + b.String() + strings.Join(warnings, "")
}

// vectorDoctorSection renders the Doctor vector block: per-engine k-NN path
// labels, per-collection embedding counts for engines with size
// introspection, and the full-scan WARN for those without.
func vectorDoctorSection(ctx context.Context, engines []Engine) string {
	var b strings.Builder

	b.WriteString("\n--- Vectors ---\n")

	reported := false

	for _, eng := range engines {
		if _, isVB := eng.(VectorBackend); !isVB {
			continue
		}

		reported = true

		if path := vectorSearchPathLabel(eng); path != "" {
			fmt.Fprintf(&b, "  %s: k-NN path %s\n", eng.Profile().Name, path)
		}
	}

	for _, eng := range engines {
		vc, counts := eng.(VectorCounter)
		if !counts {
			continue
		}

		cols, err := vc.VectorCollections(ctx)
		if err != nil {
			fmt.Fprintf(&b, "  %s: ERROR listing collections: %v\n", eng.Profile().Name, err)

			continue
		}

		for _, col := range cols {
			n, err := vc.VectorCount(ctx, col)
			if err != nil {
				fmt.Fprintf(&b, "  %s/%s: ERROR counting: %v\n", eng.Profile().Name, col, err)

				continue
			}

			fmt.Fprintf(&b, "  %s/%s: %d vectors\n", eng.Profile().Name, col, n)
		}
	}

	for _, eng := range engines {
		if _, isVB := eng.(VectorBackend); !isVB {
			continue
		}

		if _, counts := eng.(VectorCounter); counts {
			continue
		}

		fmt.Fprintf(&b, "  %s: WARN full-scan vector search (no VectorCounter)\n",
			eng.Profile().Name)
	}

	if !reported {
		b.WriteString("  no engine serves vector search\n")
	}

	return b.String()
}
