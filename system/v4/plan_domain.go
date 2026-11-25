package system

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// planDomainProjections builds the auto-projections and decoder, then runs
// the coeffect validation gate over the declared event universe.
func planDomainProjections(
	domain DomainConfig,
	safetyReport *ScreamReport,
) ([]any, eventDecoderFn, error) {
	// Indexed even without projections: a shadowed Evolution is a
	// configuration mistake whether or not something inherits from it.
	evoIndex, err := indexEvolutions(domain.Evolutions)
	if err != nil {
		return nil, nil, err
	}

	processed := []any(nil)
	decoder := eventDecoderFn(nil)
	consumed := map[event.Type][]string(nil)

	if len(domain.Projections) > 0 {
		processed, decoder, consumed, err = buildProjections(evoIndex, domain.Projections)
		if err != nil {
			return nil, nil, fmt.Errorf("system: build projections: %w", err)
		}
	}

	// Coeffect validation gate (ADR-0136 follow-up): fail composition when a
	// declared event universe exists and something consumes outside it.
	// DomainConfig.Schema types join the universe — a payload-contract
	// declaration is also a journal-universe declaration (2026-10-09 T1).
	eventUniverse := declaredEventTypes(domain)

	if len(eventUniverse) > 0 && !domain.DisableCoeffectValidation {
		if err := validateCoeffectGraph(
			eventUniverse, domain.Evolutions, consumed, safetyReport,
		); err != nil {
			return nil, nil, err
		}
	}

	return processed, decoder, nil
}

// indexEvolutions keys the Evolutions by result type. Inheritance is by
// result type, so a second Evolution for the same type would silently
// discard the first one's folds: that is refused instead.
func indexEvolutions(evolutions []EvolutionSpec) (map[reflect.Type]*evolutionSpec, error) {
	evoIndex := make(map[reflect.Type]*evolutionSpec, len(evolutions))

	for _, e := range evolutions {
		es, ok := e.(*evolutionSpec)
		if !ok {
			return nil, fmt.Errorf("system: unreachable: unknown EvolutionSpec %T", e)
		}

		if first, dup := evoIndex[es.resultType]; dup {
			return nil, fmt.Errorf(
				"system: evolutions %q and %q both declare how %s materializes; "+
					"inheritance is by result type, so %q would never be used — keep one",
				first.name, es.name, es.resultType, first.name,
			)
		}

		evoIndex[es.resultType] = es
	}

	return evoIndex, nil
}

// warnDecoderConflicts reports a wire event type that several projections
// decode into different Go types: the decoder follows the last declaration,
// so the other projections' folds get a value of the wrong type at runtime.
// A warning rather than an error because the sample structs are equal in
// every sane domain and a deliberate narrower sample is not ruled out.
func warnDecoderConflicts(entries []decoderTypeEntry) {
	first := make(map[string]reflect.Type, len(entries))

	for _, e := range entries {
		seen, ok := first[e.eventType]
		if !ok {
			first[e.eventType] = e.sampleType

			continue
		}

		if seen != e.sampleType {
			slog.Warn(
				"system: event type is declared with two different sample structs; "+
					"it decodes as the last declared one, so the other projection's fold "+
					"receives the wrong Go type",
				"event_type", e.eventType,
				"sample_types", seen.String()+" and "+e.sampleType.String(),
			)
		}
	}
}
