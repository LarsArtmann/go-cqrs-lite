module github.com/larsartmann/go-cqrs-lite/example/metaengine-quickstart

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.5.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.0.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../../id
	github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4 => ../../metaengine/projectionadapter
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ../../metaengine/sqliteengine
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../metaengine
	github.com/larsartmann/go-cqrs-lite/projection/v4 => ../../projection
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 => ../../projectionhost
	github.com/larsartmann/go-cqrs-lite/record/v4 => ../../record
)
