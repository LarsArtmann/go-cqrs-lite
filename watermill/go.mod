module github.com/larsartmann/go-cqrs-lite/watermill

go 1.26.3

require (
	github.com/ThreeDotsLabs/watermill v1.4.6
	github.com/larsartmann/go-cqrs-lite/core v1.0.0
)

require github.com/larsartmann/go-cqrs-lite/memory v1.0.0

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/larsartmann/go-error-family v0.1.1 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../core
	github.com/larsartmann/go-cqrs-lite/memory => ../memory
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
