module github.com/larsartmann/go-cqrs-lite/deriver/v3

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.0.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/command/v3 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v3 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v3 => ../id
)
