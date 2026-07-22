module github.com/larsartmann/go-cqrs-lite/stack/postgres/v4

go 1.26.4

require (
	github.com/jackc/pgx/v5 v5.10.0
	github.com/larsartmann/go-error-family v0.7.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)
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
	github.com/larsartmann/go-cqrs-lite/storage/v4 => ../../storage
)

