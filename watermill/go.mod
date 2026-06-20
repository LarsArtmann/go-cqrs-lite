module github.com/larsartmann/go-cqrs-lite/watermill/v2

go 1.26.3

require (
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.6.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.6.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v2 v2.5.0
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 v2.6.0 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v2 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../id
	github.com/larsartmann/go-cqrs-lite/query/v2 => ../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v2 => ../storage/memory
)
