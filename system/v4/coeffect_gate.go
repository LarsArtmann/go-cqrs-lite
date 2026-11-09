package system

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ErrDanglingEventSubscription is returned by New when a projection or
// evolution consumes an event type that DomainConfig.Events does not declare.
// Nearly always a typo in the coeffect specification — the kind that
// otherwise surfaces only as a projection that silently never updates.
var ErrDanglingEventSubscription = errors.New(
	"system: dangling event subscription (coeffect gate)",
)

// coeffectRuleName is the ScreamReport rule id for gate diagnostics.
const coeffectRuleName = "coeffect.unconsumed_event"

// validateCoeffectGraph checks the declared event universe (DomainConfig.Events)
// against the consumed set gathered from evolutions and projection decoder
// entries. A consumed-but-undeclared type is a hard error (typo class); a
// declared-but-unconsumed type is an advisory (dead events are legitimate for
// audit-only journals) surfaced via ScreamReport and slog.
func validateCoeffectGraph(
	declared []event.Type,
	evolutions []EvolutionSpec,
	consumed map[event.Type][]string,
	report *ScreamReport,
) error {
	declaredSet := make(map[event.Type]struct{}, len(declared))
	for _, t := range declared {
		declaredSet[t] = struct{}{}
	}

	allConsumed := make(map[event.Type][]string, len(consumed))
	maps.Copy(allConsumed, consumed)

	for eventType := range evolutionConsumedTypes(evolutions) {
		if _, ok := allConsumed[eventType]; !ok {
			allConsumed[eventType] = nil
		}
	}

	var dangling []string

	for _, eventType := range slices.Sorted(maps.Keys(allConsumed)) {
		if _, ok := declaredSet[eventType]; !ok {
			consumers := allConsumed[eventType]
			if len(consumers) == 0 {
				consumers = []string{"(evolution without projection)"}
			}

			dangling = append(dangling, fmt.Sprintf(
				"%s (consumed by: %v)", eventType, consumers,
			))
		}
	}

	if len(dangling) > 0 {
		return fmt.Errorf(
			"%w: event types consumed but not declared in DomainConfig.Events: %s "+
				"(declare them, or silence with DomainConfig.DisableCoeffectValidation)",
			ErrDanglingEventSubscription, dangling,
		)
	}

	for _, eventType := range slices.Sorted(maps.Keys(declaredSet)) {
		if _, ok := allConsumed[eventType]; ok {
			continue
		}

		detail := fmt.Sprintf(
			"event type %s is declared in DomainConfig.Events but no projection or "+
				"evolution consumes it (legitimate for audit-only journals)",
			eventType,
		)

		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierAdvisory,
			Rule:   coeffectRuleName,
			Detail: detail,
		})

		slog.Warn("system: unconsumed event type declared", "event_type", string(eventType))
	}

	return nil
}

// evolutionConsumedTypes collects every event type declared by an Evolution
// (named samples and explicit folds alike) — evolutions define fold intent
// whether or not a projection references them.
func evolutionConsumedTypes(evolutions []EvolutionSpec) map[event.Type]struct{} {
	types := make(map[event.Type]struct{})

	for _, e := range evolutions {
		es, ok := e.(*evolutionSpec)
		if !ok {
			continue
		}

		for _, s := range es.samples {
			types[event.Type(s.EventType())] = struct{}{}
		}

		for _, f := range es.explicitFolds {
			types[event.Type(f.eventType)] = struct{}{}
		}
	}

	return types
}
