module github.com/larsartmann/go-cqrs-lite/middleware

go 1.26.2

require (
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
	github.com/larsartmann/go-cqrs-lite/testhelpers v0.0.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/larsartmann/go-error-family v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../core
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
