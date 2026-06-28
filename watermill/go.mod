module github.com/larsartmann/go-cqrs-lite/watermill/v3

go 1.26.3

require (
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v3 v3.0.0
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v3 v3.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v3 v3.3.0 // indirect
	github.com/larsartmann/go-error-family v0.5.1 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v3 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v3 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v3 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v3 => ../id
	github.com/larsartmann/go-cqrs-lite/query/v3 => ../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v3 => ../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v3 => ../storage/memory
)
