module github.com/larsartmann/go-cqrs-lite/testutil/v4

go 1.26.4

require pgregory.net/rapid v1.3.0
replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v4 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
)

