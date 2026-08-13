module github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4 v4.0.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.10.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.2.0
	github.com/onsi/gomega v1.42.1
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-codec v0.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.4.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-retry v0.3.1 // indirect
	github.com/larsartmann/go-sse v0.5.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4 => ../
