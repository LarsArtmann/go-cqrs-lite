module github.com/larsartmann/go-cqrs-lite/stack/bench/v4

go 1.26.4
replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../../codec
	github.com/larsartmann/go-cqrs-lite/command/v4 => ../../command
	github.com/larsartmann/go-cqrs-lite/decider/v4 => ../../decider
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../../id
	github.com/larsartmann/go-cqrs-lite/kv/v4 => ../../kv
	github.com/larsartmann/go-cqrs-lite/otel/v4 => ../../otel
	github.com/larsartmann/go-cqrs-lite/query/v4 => ../../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../../snapshot
	github.com/larsartmann/go-cqrs-lite/stack/v4 => ../../stack
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../../storage/memory
	github.com/larsartmann/go-cqrs-lite/watermill/v4 => ../../watermill
)

