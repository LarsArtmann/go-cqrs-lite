module github.com/larsartmann/go-cqrs-lite/schema/v4

go 1.26.4

require github.com/larsartmann/go-error-family v0.7.0

require (
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.0.2
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.2
	github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.2.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.0.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.0.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v4 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
	github.com/larsartmann/go-cqrs-lite/query/v4 => ../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../storage/memory
)
