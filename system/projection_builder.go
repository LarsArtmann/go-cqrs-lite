package system

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
	name string
	build func() (queryDecl any, decoderEntries []decoderEntry, err error)
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
func buildProjections(
	decls []ProjectionDeclaration,
) (queryDecls []any, eventDecoder eventDecoderFn, err error) {
	var allEntries []decoderTypeEntry

	for _, decl := range decls {
		switch d := decl.(type) {
		case ProjectionSpec:
			queryDecl, entries, buildErr := d.build()
			if buildErr != nil {
				return nil, nil, buildErr
			}

			queryDecls = append(queryDecls, queryDecl)

			for _, e := range entries {
				allEntries = append(allEntries, decoderTypeEntry{
					eventType:  e.eventType,
					sampleType: reflect.TypeOf(e.sample),
				})
			}
		case rawQuerySpec:
			queryDecls = append(queryDecls, d.decl)
		default:
			return nil, nil, fmt.Errorf(
				"system: unreachable: unknown ProjectionDeclaration %T", decl,
			)
		}
	}

	if len(allEntries) > 0 {
		eventDecoder = buildEventDecoder(allEntries)
	}

	return queryDecls, eventDecoder, nil
}

// decoderTypeEntry pairs a wire event type with its reflect.Type.
type decoderTypeEntry struct {
	eventType  string
	sampleType reflect.Type
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
			c, err := codec.ForEncoding(codec.Encoding(evt.Encoding()))
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
