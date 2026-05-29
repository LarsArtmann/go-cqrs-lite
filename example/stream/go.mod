module github.com/larsartmann/go-cqrs-lite/example/stream

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/core v1.6.0
	github.com/larsartmann/go-cqrs-lite/memory v1.6.0
	github.com/larsartmann/go-cqrs-lite/stream v1.6.0
)

require (
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../../core
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
	github.com/larsartmann/go-cqrs-lite/stream => ../../stream
)
