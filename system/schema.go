package system

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// applySchemaDeclaration compiles the DomainConfig.Schema declaration
// (2026-10-09 T1) and decorates the event store with the compiled upcast
// chain. Every read path built on sys.eventStore — decider loads via
// RegisterDecider and the projection host's journal — then sees CURRENT
// payloads regardless of what schema version was stored. Writes pass
// through unchanged (ADR-0126 capability preservation: the decorated store
// keeps Journal, SeekableJournal, and Closer).
//
// The declaration is DATA in DomainConfig, never a second registry: New is
// the only composition point (ADR-0123), and schema.Declare validation runs
// at boot — bad declarations fail composition, not the first read.
func applySchemaDeclaration(sys *System, domain DomainConfig) error {
	if len(domain.Schema) == 0 {
		return nil
	}

	chain, err := schema.Declare(domain.Schema...)
	if err != nil {
		return fmt.Errorf("system: declare schema: %w", err)
	}

	sys.eventStore = event.DecorateStore(sys.eventStore, nil, chain.SourceTransform())

	return nil
}

// declaredEventTypes coalesces the coeffect gate's event universe: explicit
// DomainConfig.Events plus every type named by DomainConfig.Schema
// (deduplicated). A Schema declaration IS a declaration of the journal's
// event universe, so its types count for the dangling-subscription check.
func declaredEventTypes(domain DomainConfig) []event.Type {
	if len(domain.Schema) == 0 {
		return domain.Events
	}

	seen := make(map[event.Type]struct{}, len(domain.Events)+len(domain.Schema))
	merged := make([]event.Type, 0, len(domain.Events)+len(domain.Schema))

	for _, declared := range append(append([]event.Type{}, domain.Events...), schemaTypes(domain.Schema)...) {
		//art-dupl:accept cross-module 6-line map-dedup idiom (pairs with cqrs-lint feature_profile_stores.go) — restructuring to dodge the detector would be worse code
		if _, duplicate := seen[declared]; duplicate {
			continue
		}

		seen[declared] = struct{}{}
		merged = append(merged, declared)
	}

	return merged
}

func schemaTypes(declarations []schema.EventSchema) []event.Type {
	types := make([]event.Type, 0, len(declarations))

	for _, declared := range declarations {
		types = append(types, declared.Type())
	}

	return types
}
