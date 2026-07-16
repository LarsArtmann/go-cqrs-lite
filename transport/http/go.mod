module github.com/larsartmann/go-cqrs-lite/transport/http/v4

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testutil/v4 v4.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
)

require github.com/larsartmann/go-cqrs-lite/query/v4 v4.0.0-00010101000000-000000000000 // indirect

require (
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.0.0-00010101000000-000000000000 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.2
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-error-family v0.7.0
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../../event/v4/eventtest

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../../codec
	github.com/larsartmann/go-cqrs-lite/command/v4 => ../../command
	github.com/larsartmann/go-cqrs-lite/dedup/v4 => ../../dedup
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../../id
	github.com/larsartmann/go-cqrs-lite/otel/v4 => ../../otel
	github.com/larsartmann/go-cqrs-lite/query/v4 => ../../query
	github.com/larsartmann/go-cqrs-lite/schema/v4 => ../../schema
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../../storage/memory
	github.com/larsartmann/go-cqrs-lite/testutil/v4 => ../../testutil
)

replace github.com/larsartmann/go-cqrs-lite/metadata/v4 => ../../metadata
