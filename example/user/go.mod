module github.com/larsartmann/go-cqrs-lite/example/user

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/catalog v1.6.0
	github.com/larsartmann/go-cqrs-lite/codec v1.6.0
	github.com/larsartmann/go-cqrs-lite/command v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/decider v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event v1.6.0
	github.com/larsartmann/go-cqrs-lite/id v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/memory v1.6.0
	github.com/larsartmann/go-cqrs-lite/middleware v1.6.0
	github.com/larsartmann/go-cqrs-lite/query v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/signing v1.6.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel v1.6.0 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/catalog => ../../catalog
	github.com/larsartmann/go-cqrs-lite/codec => ../../codec
	github.com/larsartmann/go-cqrs-lite/command => ../../command
	github.com/larsartmann/go-cqrs-lite/decider => ../../decider
	github.com/larsartmann/go-cqrs-lite/event => ../../event
	github.com/larsartmann/go-cqrs-lite/id => ../../id
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
	github.com/larsartmann/go-cqrs-lite/middleware => ../../middleware
	github.com/larsartmann/go-cqrs-lite/otel => ../../otel
	github.com/larsartmann/go-cqrs-lite/query => ../../query
	github.com/larsartmann/go-cqrs-lite/signing => ../../signing
)
