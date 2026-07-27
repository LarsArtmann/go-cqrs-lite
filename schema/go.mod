module github.com/larsartmann/go-cqrs-lite/schema/v4

go 1.26.5

require github.com/larsartmann/go-error-family v0.10.0

require (
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.1.1
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.1.0
	github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.3.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.1.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.1.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.3 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)
