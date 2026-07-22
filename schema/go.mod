module github.com/larsartmann/go-cqrs-lite/schema/v4

go 1.26.4

require github.com/larsartmann/go-error-family v0.7.0

require pgregory.net/rapid v1.3.0
replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v4 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
	github.com/larsartmann/go-cqrs-lite/query/v4 => ../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../storage/memory
)

