module github.com/larsartmann/go-cqrs-lite/watermill

go 1.26.3

require (
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/larsartmann/go-cqrs-lite/event v1.7.1
	github.com/larsartmann/go-cqrs-lite/id v1.7.1
	github.com/larsartmann/go-cqrs-lite/memory v1.7.1
	github.com/larsartmann/go-cqrs-lite/testhelpers v1.7.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/command v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/query v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot v1.7.1 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/event => ../event
	github.com/larsartmann/go-cqrs-lite/id => ../id
	github.com/larsartmann/go-cqrs-lite/memory => ../memory
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
