// Package metaengine carries the typed stack↔metaengine integration: the
// registration option and the concrete-store discovery that the root stack
// package deliberately cannot express without importing metaengine (issue
// #36; the root package holds only the [stack.MetaEngineStore] lifecycle
// seam).
//
// The whole stack family is Deprecated for removal at v5 (ADR-0123) —
// system.New is the single composition root. This module exists so v4.x
// consumers of plain stack composition carry no metaengine dependency unless
// they opt into it here.
package metaengine

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// WithStore registers a *metaengine.Store on the Bundle for lifecycle
// management: Bundle.Close releases the store's engine resources (SQLite
// connections, Pebble handles, …). The typed companion to the deprecated
// [stack.WithMetaEngine].
//
// The consumer constructs the Store via metaengine.Plan(engines, queries…)
// (typed generics) and passes it here; benchkit auto-discovers it through
// the bundle unless Config.SkipMetaEngine is set.
func WithStore(store *metaengine.Store) stack.Option {
	return stack.WithMetaEngine(store)
}

// Store returns the *metaengine.Store registered on the Bundle, or nil when
// none was set (the [stack.Bundle.MetaEngine] seam is type-erased; this
// helper performs the recovery). The typed companion to the deprecated
// [stack.Bundle.MetaEngine].
func Store(b *stack.Bundle) *metaengine.Store {
	if s, ok := b.MetaEngine().(*metaengine.Store); ok {
		return s
	}
	return nil
}
