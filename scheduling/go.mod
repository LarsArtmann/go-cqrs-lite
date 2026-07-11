module github.com/larsartmann/go-cqrs-lite/scheduling/v3

go 1.26.4

replace github.com/larsartmann/go-cqrs-lite/id/v3 => ../id

require github.com/larsartmann/go-cqrs-lite/testutil/v3 v3.0.0-00010101000000-000000000000

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-error-family v0.7.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/testutil/v3 => ../testutil

replace github.com/larsartmann/go-cqrs-lite/codec/v3 => ../codec

replace github.com/larsartmann/go-cqrs-lite/command/v3 => ../command

replace github.com/larsartmann/go-cqrs-lite/metadata/v3 => ../metadata

replace github.com/larsartmann/go-cqrs-lite/dispatcher/v3 => ../dispatcher

replace github.com/larsartmann/go-cqrs-lite/event/v3 => ../event

replace github.com/larsartmann/go-cqrs-lite/event/v3/eventtest => ../event/v3/eventtest

replace github.com/larsartmann/go-cqrs-lite/snapshot/v3 => ../snapshot
