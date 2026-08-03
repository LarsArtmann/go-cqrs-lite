module github.com/larsartmann/go-cqrs-lite/idempotency/v4

go 1.26.5

require (
	github.com/larsartmann/go-error-family v0.10.0
	github.com/larsartmann/go-idempotency v0.0.0-00010101000000-000000000000
	pgregory.net/rapid v1.3.0
)

replace github.com/larsartmann/go-idempotency => ../../go-idempotency
