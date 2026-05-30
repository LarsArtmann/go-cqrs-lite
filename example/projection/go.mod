module github.com/larsartmann/go-cqrs-lite/example/projection

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/codec v1.7.1
	github.com/larsartmann/go-cqrs-lite/event v1.7.1
	github.com/larsartmann/go-cqrs-lite/id v1.7.1
	github.com/larsartmann/go-cqrs-lite/memory v1.7.1
	github.com/larsartmann/go-cqrs-lite/projection v1.7.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/otel v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot v1.7.1 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec => ../../codec
	github.com/larsartmann/go-cqrs-lite/event => ../../event
	github.com/larsartmann/go-cqrs-lite/id => ../../id
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
	github.com/larsartmann/go-cqrs-lite/otel => ../../otel
	github.com/larsartmann/go-cqrs-lite/projection => ../../projection
)
