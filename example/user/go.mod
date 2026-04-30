module github.com/larsartmann/go-cqrs-lite/example/user

go 1.26.2

require (
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
	github.com/larsartmann/go-cqrs-lite/memory v0.0.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../../core
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
)
