module github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4

go 1.26.4

require (
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/larsartmann/go-error-family v0.7.0
	modernc.org/sqlite v1.54.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mattn/go-isatty v0.0.23 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.47.0 // indirect
	modernc.org/libc v1.74.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
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

