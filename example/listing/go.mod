module github.com/larsartmann/go-cqrs-lite/example/listing

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/core v1.6.0
	github.com/larsartmann/go-cqrs-lite/listing v1.6.0
	github.com/larsartmann/go-cqrs-lite/memory v1.6.0
)

require (
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec v0.0.0-20260529144800-51ed93d67be5 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../../core
	github.com/larsartmann/go-cqrs-lite/listing => ../../listing
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
)
