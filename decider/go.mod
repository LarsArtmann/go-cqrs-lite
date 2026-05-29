module github.com/larsartmann/go-cqrs-lite/decider

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/event v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/otel v1.6.0
	github.com/larsartmann/go-cqrs-lite/testhelpers v1.6.0
	github.com/onsi/ginkgo/v2 v2.29.0
	github.com/onsi/gomega v1.41.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/event => ../event
	github.com/larsartmann/go-cqrs-lite/id => ../id
	github.com/larsartmann/go-cqrs-lite/otel => ../otel
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
