module github.com/larsartmann/go-cqrs-lite/example/graph-demo

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/graph/v3 v3.3.0
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.3.0
)

require (
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v3 v3.0.0-20260628225605-630e04ca84a8 // indirect
	github.com/larsartmann/go-error-family v0.5.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/event/v3 => ../../event
	github.com/larsartmann/go-cqrs-lite/graph/v3 => ../../graph
	github.com/larsartmann/go-cqrs-lite/id/v3 => ../../id
)
