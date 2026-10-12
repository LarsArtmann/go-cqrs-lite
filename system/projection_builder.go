package system

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// LookupInput is the query input type for point-lookup projections.
// K is typically string or a branded ID type.
type LookupInput[K any] struct {
	ID K
}

// ScanInput is the query input type for filtered-scan projections (QuerySet).
type ScanInput struct{}

// CountInput is the query input type for counter projections.
type CountInput struct{}

// ProjectionDeclaration is a sealed interface for projection/query
// declarations. Only constructors in the system package can produce values
// that satisfy it. This replaces the previous []any slice with compile-time
// type safety: stray strings, nils, or typos are rejected at build time.
type ProjectionDeclaration interface {
	isProjectionDeclaration()
}

// ProjectionSpec is a type-erased projection declaration created by [Lookup],
// [QuerySet], or [Count]. It carries a build closure that generates the query
// declaration and decoder entries when system.New() processes it.
type ProjectionSpec struct {
	name       string
	resultType reflect.Type
	build      func(evoIndex map[reflect.Type]*evolutionSpec) (queryDecl any, decoderEntries []decoderEntry, err error)
}

func (ProjectionSpec) isProjectionDeclaration() {}

// rawQuerySpec wraps a raw metaengine.QueryDecl for passthrough to Plan().
type rawQuerySpec struct {
	decl any
}

func (rawQuerySpec) isProjectionDeclaration() {}

// RawQuery wraps a raw metaengine.QueryDecl so it can be used in
// DomainConfig.Projections alongside auto-generated projections.
func RawQuery(decl any) ProjectionDeclaration {
	return rawQuerySpec{decl: decl}
}

// decoderEntry pairs a wire event type with its sample struct for decoder
// construction.
type decoderEntry struct {
	eventType string
	sample    any
}

// eventDecoderFn matches projectionadapter.EventDecoder.
type eventDecoderFn func(evt event.Event) (any, error)

// buildProjections processes ProjectionDeclaration values from
// DomainConfig.Projections. It type-switches on the sealed interface to
// separate auto-generated ProjectionSpec values from raw QueryDecl passthroughs.
// It also returns the consumed event-type set (event type → consuming
// projection names) for the coeffect validation gate.
func buildProjections(
	evolutions []EvolutionSpec,
	decls []ProjectionDeclaration,
) (queryDecls []any, eventDecoder eventDecoderFn, consumed map[event.Type][]string, err error) {
	evoIndex := make(map[reflect.Type]*evolutionSpec)
	for _, e := range evolutions {
		es, ok := e.(*evolutionSpec)
		if !ok {
			return nil, nil, nil, fmt.Errorf(
				"system: unreachable: unknown EvolutionSpec %T", e,
			)
		}

		evoIndex[es.resultType] = es
	}

	var allEntries []decoderTypeEntry
	consumed = make(map[event.Type][]string)

	for _, decl := range decls {
		switch d := decl.(type) {
		case ProjectionSpec:
			queryDecl, entries, buildErr := d.build(evoIndex)
			if buildErr != nil {
				return nil, nil, nil, buildErr
			}

			queryDecls = append(queryDecls, queryDecl)

			for _, e := range entries {
				allEntries = append(allEntries, decoderTypeEntry{
					eventType:  e.eventType,
					sampleType: reflect.TypeOf(e.sample),
				})

				consumed[event.Type(e.eventType)] = append(
					consumed[event.Type(e.eventType)],
					fmt.Sprintf("projection %q", d.name),
				)
			}
		case rawQuerySpec:
			queryDecls = append(queryDecls, d.decl)

			// RawQuery passthroughs still consume their folds' event types.
			// Without this the coeffect gate treats every RawQuery-only domain
			// event as a dangling subscription even though its own projection
			// consumes it.
			name, foldTypes := rawQueryConsumedTypes(d.decl)
			for _, et := range foldTypes {
				consumed[event.Type(et)] = append(
					consumed[event.Type(et)],
					fmt.Sprintf("projection %q", name),
				)
			}
		default:
			return nil, nil, nil, fmt.Errorf(
				"system: unreachable: unknown ProjectionDeclaration %T", decl,
			)
		}
	}

	if len(allEntries) > 0 {
		eventDecoder = buildEventDecoder(allEntries)
	}

	return queryDecls, eventDecoder, consumed, nil
}

// decoderTypeEntry pairs a wire event type with its reflect.Type.
type decoderTypeEntry struct {
	eventType  string
	sampleType reflect.Type
}

// rawQueryConsumedTypes extracts the wire event types a raw
// metaengine.QueryDecl consumes via its folds, plus its query name for the
// coeffect gate's consumer labels. Best-effort: passthroughs that are not
// QueryDecl-shaped contribute nothing.
func rawQueryConsumedTypes(decl any) (name string, types []string) {
	folds, ok := decl.(interface{ QueryFolds() []metaengine.Fold })
	if !ok {
		return "<raw query>", nil
	}

	name = "<raw query>"
	if n, named := decl.(interface{ QueryName() string }); named {
		name = n.QueryName()
	}

	seen := make(map[string]struct{})
	for _, f := range folds.QueryFolds() {
		et := f.EventType()
		if _, dup := seen[et]; dup {
			continue
		}

		seen[et] = struct{}{}
		types = append(types, et)
	}

	return name, types
}

// buildEventDecoder creates an EventDecoder that auto-detects the payload
// encoding (JSON or CBOR) and decodes into the correct Go type.
func buildEventDecoder(entries []decoderTypeEntry) eventDecoderFn {
	typeMap := make(map[string]reflect.Type, len(entries))
	for _, e := range entries {
		typeMap[e.eventType] = e.sampleType
	}

	return func(evt event.Event) (any, error) {
		eventType := string(evt.Type())

		t, ok := typeMap[eventType]
		if !ok {
			return nil, fmt.Errorf("system: no decoder registered for event type %q", eventType)
		}

		val := reflect.New(t).Interface()

		if len(evt.Payload()) > 0 {
			c, err := codec.ForEncoding(evt.Encoding())
			if err != nil {
				return nil, fmt.Errorf("system: codec for encoding %q: %w", evt.Encoding(), err)
			}

			if err := c.Decode(evt.Payload(), val); err != nil {
				return nil, fmt.Errorf("system: decode %s: %w", eventType, err)
			}
		}

		return reflect.ValueOf(val).Elem().Interface(), nil
	}
}
