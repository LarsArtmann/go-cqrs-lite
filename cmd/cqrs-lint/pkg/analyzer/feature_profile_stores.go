package analyzer

import "slices"

// addStore records a detected persistence-backend signal. The primary Store
// keeps its historical first-wins selection semantics (assigned by the
// detection passes, not here); Stores accumulates every distinct signal so
// mixed pools (journal backend + projection engines) stay visible.
func (fp *FeatureProfile) addStore(kind StoreKind) {
	if kind == StoreUnknown || kind == StoreNone {
		return
	}
	if !slices.Contains(fp.Stores, kind) {
		fp.Stores = append(fp.Stores, kind)
	}
}

// refineStore replaces a previously recorded signal: the AST refinement
// pass resolves a generic "custom" signal (a storage/ import) into the
// concrete backend its constructor names. The refined kind replaces the
// signal it refines; primary Store is updated by the caller.
func (fp *FeatureProfile) refineStore(from, to StoreKind) {
	if to == StoreUnknown || to == StoreNone {
		return
	}
	for i, kind := range fp.Stores {
		if kind == from {
			fp.Stores[i] = to
			break
		}
	}
	fp.addStore(to)
	fp.dedupeStores()
}

func (fp *FeatureProfile) dedupeStores() {
	seen := make(map[StoreKind]struct{}, len(fp.Stores))
	kept := fp.Stores[:0]
	for _, kind := range fp.Stores {
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		kept = append(kept, kind)
	}
	fp.Stores = kept
}

// EffectiveStores returns every meaningful detected-or-configured backend
// kind: the Stores union when populated, otherwise the legacy primary.
// Unknown and None are filtered — they mean "no signal", not "a backend".
// Rules that quantify over backends ("is ANY store SQL-backed?") must use
// this list, never the primary alone: a mixed pool (postgres journal +
// memory projections) is SQL-backed even though its primary alone might
// suggest otherwise.
func (fp FeatureProfile) EffectiveStores() []StoreKind {
	kinds := fp.Stores
	if len(kinds) == 0 && fp.Store != StoreUnknown && fp.Store != StoreNone {
		kinds = []StoreKind{fp.Store}
	}
	return slices.DeleteFunc(slices.Clone(kinds), func(kind StoreKind) bool {
		return kind == StoreUnknown || kind == StoreNone
	})
}

// AnyStoreSQL reports whether any backend can push down filters/sorts
// (ORDER BY / WHERE). Gates pushdown-adoption suggestions (F022-F025).
func (fp FeatureProfile) AnyStoreSQL() bool {
	for _, kind := range fp.EffectiveStores() {
		if kind.IsSQL() {
			return true
		}
	}
	return false
}

// AnyStoreDistributed reports whether any backend runs as a separate
// server process, enabling (and requiring) multi-instance deployment.
func (fp FeatureProfile) AnyStoreDistributed() bool {
	for _, kind := range fp.EffectiveStores() {
		if kind.IsDistributed() {
			return true
		}
	}
	return false
}

// AnyStorePersistent reports whether any backend survives restarts — the
// complement of an all-in-memory (or no-backend) setup. Used by rules that
// flag in-memory per-call stores as inconsistent with an otherwise
// persistent system (C017) or skip when nothing persists at all (C036).
func (fp FeatureProfile) AnyStorePersistent() bool {
	for _, kind := range fp.EffectiveStores() {
		if kind != StoreMemory && kind != StoreUnknown && kind != StoreNone {
			return true
		}
	}
	return false
}
