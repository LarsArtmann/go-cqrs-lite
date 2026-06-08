package projection

import (

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer("projection")
}

func projectionAttrs(name string) []cqrsotel.KeyValue {
	return []cqrsotel.KeyValue{
		cqrsotel.AttrString(cqrsotel.AttrProjectionName, name),
	}
}
