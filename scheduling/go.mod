module github.com/larsartmann/go-cqrs-lite/scheduling/v4

go 1.26.4

replace github.com/larsartmann/go-cqrs-lite/id/v4 => ../id

require github.com/larsartmann/go-cqrs-lite/testutil/v4 v4.0.0

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.0.0 // indirect
	github.com/larsartmann/go-error-family v0.7.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/testutil/v4 => ../testutil

replace github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec

replace github.com/larsartmann/go-cqrs-lite/command/v4 => ../command

replace github.com/larsartmann/go-cqrs-lite/metadata/v4 => ../metadata

replace github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher

replace github.com/larsartmann/go-cqrs-lite/event/v4 => ../event

replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../event/v4/eventtest

replace github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot
