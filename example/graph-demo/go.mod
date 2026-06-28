module github.com/larsartmann/go-cqrs-lite/example/graph-demo

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/graph/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.3.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/event/v3 => ../../event
	github.com/larsartmann/go-cqrs-lite/graph/v3 => ../../graph
	github.com/larsartmann/go-cqrs-lite/id/v3 => ../../id
)
