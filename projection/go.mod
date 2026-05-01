module github.com/larsartmann/go-cqrs-lite/projection

go 1.26.2

require (
	github.com/cockroachdb/errors v1.12.0
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
	github.com/larsartmann/go-cqrs-lite/memory v0.0.0
	github.com/samber/ro v0.3.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../core
	github.com/larsartmann/go-cqrs-lite/memory => ../memory
)
