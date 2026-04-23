module github.com/larsartmann/go-cqrs-lite/catalog

go 1.26

require (
	github.com/go-faster/yaml v0.4.6
	github.com/go-json-experiment/json v0.0.0-20260214004413-d219187c3433
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
)

replace github.com/larsartmann/go-cqrs-lite/core => ../core
