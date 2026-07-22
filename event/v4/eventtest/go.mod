module github.com/larsartmann/go-cqrs-lite/event/v4/eventtest

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.2
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.0.1
	github.com/larsartmann/go-error-family v0.7.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)
replace (
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../../../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../../../id
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../../../snapshot
)

