module github.com/larsartmann/go-cqrs-lite/example/saga-pattern

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/memory/v2 v2.0.0
	github.com/larsartmann/go-cqrs-lite/projection/v2 v2.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v2 v2.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 v2.0.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../../codec
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../../id
	github.com/larsartmann/go-cqrs-lite/memory/v2 => ../../memory
	github.com/larsartmann/go-cqrs-lite/otel/v2 => ../../otel
	github.com/larsartmann/go-cqrs-lite/projection/v2 => ../../projection
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../../snapshot
)
