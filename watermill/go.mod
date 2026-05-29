module github.com/larsartmann/go-cqrs-lite/watermill

go 1.26.3

require (
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/larsartmann/go-cqrs-lite/event v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id v0.0.0-00010101000000-000000000000
)

require (
	github.com/larsartmann/go-cqrs-lite/memory v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testhelpers v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
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
