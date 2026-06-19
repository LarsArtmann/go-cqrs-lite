module github.com/larsartmann/go-cqrs-lite/stack/postgres/v2

go 1.26.3

require (
	github.com/jackc/pgx/v5 v5.7.1
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.5.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.5.0
	github.com/larsartmann/go-cqrs-lite/memory/v2 v2.5.0
	github.com/larsartmann/go-cqrs-lite/stack/v2 v2.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/v2 v2.5.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/decider/v2 v2.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v2 v2.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/listing/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v2 v2.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/readmodel/v2 v2.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 v2.5.0 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../../codec
	github.com/larsartmann/go-cqrs-lite/command/v2 => ../../command
	github.com/larsartmann/go-cqrs-lite/decider/v2 => ../../decider
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../../id
	github.com/larsartmann/go-cqrs-lite/kv/v2 => ../../kv
	github.com/larsartmann/go-cqrs-lite/memory/v2 => ../../memory
	github.com/larsartmann/go-cqrs-lite/otel/v2 => ../../otel
	github.com/larsartmann/go-cqrs-lite/projection/v2 => ../../projection
	github.com/larsartmann/go-cqrs-lite/query/v2 => ../../query
	github.com/larsartmann/go-cqrs-lite/readmodel/v2 => ../../readmodel
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../../snapshot
	github.com/larsartmann/go-cqrs-lite/stack/v2 => ..
	github.com/larsartmann/go-cqrs-lite/storage/v2 => ../../storage
)
