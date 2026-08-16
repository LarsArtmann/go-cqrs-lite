package metaengine

// LayoutOption represents a physical storage layout for a projection entry
// (ADR-0124 Layer 4). The planner scores each option against the engine's cost
// profile and the operator's priority to select the optimal layout.
type LayoutOption string

const (
	// LayoutEmbed stores the entire aggregate (including child collections) as
	// a single value. Optimal for read-whole-aggregate patterns on KV engines.
	// Write amplification on child mutations (read-modify-write the parent).
	LayoutEmbed LayoutOption = "Embed"

	// LayoutNormalize splits child collections into separate tables/collections
	// with foreign keys. Optimal for SQL engines (JOIN is native) and
	// append-heavy child patterns (O(1) insert into child table).
	LayoutNormalize LayoutOption = "Normalize"

	// LayoutHybrid embeds small children and normalizes large ones. The
	// threshold is operator-configurable. Future work — not yet scored.
	LayoutHybrid LayoutOption = "Hybrid"
)

// LayoutCost estimates the read, write, and storage cost of a layout option on
// a specific engine type. Costs are relative multipliers (1.0 = baseline).
type LayoutCost struct {
	Option      LayoutOption
	ReadCost    float64 // relative read cost (1.0 = baseline)
	WriteCost   float64 // relative write cost (1.0 = baseline)
	StorageCost float64 // relative storage cost (1.0 = baseline)
}

// ScoreWeighted applies priority weights to the layout cost and returns a
// single comparable score. Lower is better.
func (lc LayoutCost) ScoreWeighted(w PriorityWeights) float64 {
	return w.ReadW*lc.ReadCost + w.WriteW*lc.WriteCost + w.StorageW*lc.StorageCost
}

// scoreEmbed returns the relative cost of embedding on the given storage layout.
// KV/LSM engines favor embedding (native single-key lookup).
//
// Two constant conventions coexist:
//
//   - KV/LSM (anchor convention): the embed row is the baseline per dimension
//     (read: the cheapest op on the engine; write: 1.0; storage: the
//     3-projection duplication penalty) and normalize values are measured
//     normalize/embed ratios scaled off those anchors.
//   - Row/Columnar (geomean-centered): every embed x normalize pair multiplies
//     to 1.0 per dimension, so only the measured ratio drives decisions.
//
// KV values are CALIBRATED 2026-08-16 via BenchmarkLayoutCalibration_* on the
// memory engine (AMD Ryzen AI MAX+ 395, size-stable benches, medians of 10
// runs; the pre-2026-08-15 benches appended a child per iteration, so values
// grew unboundedly and drifted mid-run). Measured normalize/embed ratios:
// read 1.80x, write 0.84x, storage 0.485x (3-projection model).
//
// LSM values are CALIBRATED 2026-08-16 via BenchmarkDiskLayoutCalibration_* in
// metaengine/bench on real on-disk Pebble and bbolt databases (size-stable;
// the old bench asserted a typed value MapUpdate never produces, so its
// mutation silently no-oped). Measured normalize/embed ratios:
//
//	read  1.59x (geomean across Pebble 1.66x and bbolt 1.52x)
//	write 0.56x (geomean across Pebble 0.32x and bbolt 1.00x; bbolt's
//	             single-writer model neutralizes normalize's write advantage)
//
// READ values are floor-adjusted, not raw ratios. Honest values (KV 0.90 =
// 0.5x1.80, LSM 1.18 = 0.74x1.59) would flip KV/LSM Balanced and KV ReadSpeed
// to Normalize: the 3-projection storage model carries weight 1.0 under those
// priorities and overrides the measured 1.6-1.8x embed read advantage. The
// read constants are pinned at the minimum that keeps the designed lever
// matrix (KV/LSM Balanced and ReadSpeed resolve to Embed) with a >= 0.10
// margin, which also keeps decisions stable across machines (honest margins
// of ~0.05 sit within cross-machine bench variance). This lever-pinning
// tradeoff was first adopted 2026-08-11 and is retained deliberately.
//
// Row values are CALIBRATED 2026-08-15 via BenchmarkRowLayoutCalibration_* in
// metaengine/bench on file-backed SQLite, Postgres 16 (ephemeral local), and
// MySQL (QEMU port-forward; ratios corroborate Postgres). Embed is measured
// through the engine MapBackend API (meta_map JSON column); normalize against
// dedicated parent/child tables (LEFT JOIN read, O(1) child insert). Measured
// normalize/embed ratios (geomean sqlite 1.95/1.00/1.06 read, 0.66/0.38/0.56
// write, 0.33/0.33/0.41 storage):
//
//	read  1.27x   write 0.52x   storage 0.35x
//
// Columnar values are CALIBRATED 2026-08-15 via
// BenchmarkColumnarLayoutCalibration_* on file-backed DuckDB; a literal
// -benchtime=60s confirmation run reproduced the ratios within 2%
// (read 2.66x, write 0.19x, storage unchanged). Measured normalize/embed
// ratios:
//
//	read  2.62x (point lookups on OLAP engine favor embed strongly)
//	write 0.20x (DuckDB row UPDATE costs ~5x an insert)
//	storage 0.59x (columnar compression absorbs embed duplication)
func scoreEmbed(layout StorageLayout) LayoutCost {
	switch layout {
	case LayoutKV:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.5, // single key lookup — very fast
			WriteCost:   1.0, // single write (child mutations need RMW but baseline is O(1))
			StorageCost: 1.3, // data duplication across projections
		}
	case LayoutLSM:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.74, // single point read — cheapest op on LSM/B+Tree
			WriteCost:   1.10, // read-modify-write parent on child mutation
			StorageCost: 1.15, // aggregate duplicated across projections
		}
	case LayoutRow:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.89, // JSON column read — measured (Row read ratio 1.27x)
			WriteCost:   1.39, // rewrite entire JSON column on child mutation — measured 0.52x inverse
			StorageCost: 1.68, // JSON duplication across 3 projections — measured 0.35x inverse
		}
	case LayoutColumnar:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.62, // embedded nested value — DuckDB point reads favor embed (2.62x)
			WriteCost:   2.23, // UPDATE on columnar engine — measured 5x an insert
			StorageCost: 1.30, // compression absorbs most duplication (0.59x)
		}
	default:
		return LayoutCost{Option: LayoutEmbed, ReadCost: 1.0, WriteCost: 1.0, StorageCost: 1.0}
	}
}

// scoreNormalize returns the relative cost of normalizing on the given storage
// layout. SQL engines favor normalization (native JOIN + index-backed).
//
// KV Normalize values are CALIBRATED from BenchmarkLayoutCalibration_* (memory
// engine, 2026-08-11). See scoreEmbed for the LSM calibration provenance.
func scoreNormalize(layout StorageLayout) LayoutCost {
	switch layout {
	case LayoutKV:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    1.8,  // multi-key lookup + in-memory merge — calibrated from 2.2x measured ratio
			WriteCost:   0.48, // single insert into child collection — calibrated from 2.1x measured ratio
			StorageCost: 0.63, // no data duplication — calibrated from 2.06x measured ratio (3 projections)
		}
	case LayoutLSM:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    1.45, // index seek + prefix scan + decode on disk
			WriteCost:   0.75, // single key insert (no RMW)
			StorageCost: 0.80, // one copy of each fact
		}
	case LayoutRow:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    1.13, // LEFT JOIN read — measured ≈ par with JSON row on server engines
			WriteCost:   0.72, // single row insert into child table — measured
			StorageCost: 0.59, // one copy of each fact — measured
		}
	case LayoutColumnar:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    1.62, // long/narrow child table — point reads pay 2.62x vs embed
			WriteCost:   0.45, // O(1) insert vs row UPDATE — measured 0.20x ratio
			StorageCost: 0.77, // columnar layout of child tables — measured 0.59x
		}
	default:
		return LayoutCost{Option: LayoutNormalize, ReadCost: 1.0, WriteCost: 1.0, StorageCost: 1.0}
	}
}

// ScoreLayouts returns the cost of embed and normalize for the given engine
// profile. The planner selects the layout with the lowest weighted score.
func ScoreLayouts(profile EngineProfile) []LayoutCost {
	layout := defaultStorageLayout(profile)

	return []LayoutCost{
		scoreEmbed(layout),
		scoreNormalize(layout),
	}
}

// SelectLayout picks the best layout option for an engine profile given the
// operator's priority. Returns the selected option and its cost.
func SelectLayout(profile EngineProfile, priority Priority) (LayoutOption, LayoutCost) {
	costs := ScoreLayouts(profile)
	w := priority.Weights()

	best := costs[0]
	bestScore := best.ScoreWeighted(w)

	for _, c := range costs[1:] {
		score := c.ScoreWeighted(w)
		if score < bestScore {
			best = c
			bestScore = score
		}
	}

	return best.Option, best
}

// defaultStorageLayout infers the storage layout category from an engine profile.
// This maps the engine's Layouts map to a single representative layout for
// cost scoring.
func defaultStorageLayout(profile EngineProfile) StorageLayout {
	// Check the profile's layout map for a representative layout
	for _, layout := range profile.Layouts {
		switch layout {
		case LayoutKV, LayoutLSM, LayoutRow, LayoutColumnar:
			return layout
		}
	}

	// Infer from ADT support if no explicit layout
	if _, ok := profile.Supports[ADTGraph]; ok && len(profile.Supports) == 1 {
		return LayoutKV // graph engines are typically KV-like
	}

	// Default: KV (safest assumption for memory/pebble/bbolt)
	return LayoutKV
}
