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
func scoreEmbed(layout StorageLayout) LayoutCost {
	switch layout {
	case LayoutKV, LayoutLSM:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.5, // single key lookup — very fast
			WriteCost:   1.0, // single write (child mutations need RMW but baseline is O(1))
			StorageCost: 1.3, // data duplication across projections
		}
	case LayoutRow:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.7, // JSON column read — fast but loses child queryability
			WriteCost:   1.5, // rewrite entire JSON column on child mutation
			StorageCost: 1.2,
		}
	case LayoutColumnar:
		return LayoutCost{
			Option:      LayoutEmbed,
			ReadCost:    0.6, // nested/repeated column — fast for analytics
			WriteCost:   1.3,
			StorageCost: 1.1,
		}
	default:
		return LayoutCost{Option: LayoutEmbed, ReadCost: 1.0, WriteCost: 1.0, StorageCost: 1.0}
	}
}

// scoreNormalize returns the relative cost of normalizing on the given storage
// layout. SQL engines favor normalization (native JOIN + index-backed).
func scoreNormalize(layout StorageLayout) LayoutCost {
	switch layout {
	case LayoutKV, LayoutLSM:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    2.0, // multi-key lookup + in-memory merge — expensive on KV
			WriteCost:   0.5, // single insert into child collection — O(1)
			StorageCost: 0.7, // no data duplication
		}
	case LayoutRow:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    0.8, // JOIN is native — efficient
			WriteCost:   0.6, // single row insert into child table
			StorageCost: 0.8,
		}
	case LayoutColumnar:
		return LayoutCost{
			Option:      LayoutNormalize,
			ReadCost:    1.0, // long/narrow child table — fine for analytics
			WriteCost:   0.7,
			StorageCost: 0.8,
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
