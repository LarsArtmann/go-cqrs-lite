module github.com/larsartmann/go-cqrs-lite/testhelpers

go 1.26.2

require github.com/larsartmann/go-cqrs-lite/core v1.4.0

require (
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/larsartmann/go-error-family v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/core => ../core
