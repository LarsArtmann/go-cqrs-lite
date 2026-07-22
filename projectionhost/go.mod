module github.com/larsartmann/go-cqrs-lite/projectionhost/v4

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	modernc.org/sqlite v1.54.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/mattn/go-isatty v0.0.23 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.74.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

require (
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/schema/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testutil/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-error-family v0.7.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
	github.com/larsartmann/go-cqrs-lite/dedup/v4 => ../dedup
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
	github.com/larsartmann/go-cqrs-lite/projection/v4 => ../projection
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../storage/memory
)

replace github.com/larsartmann/go-cqrs-lite/testutil/v4 => ../testutil

replace github.com/larsartmann/go-cqrs-lite/storage/v4 => ../storage

replace github.com/larsartmann/go-cqrs-lite/command/v4 => ../command

replace github.com/larsartmann/go-cqrs-lite/metadata/v4 => ../metadata

replace github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher

replace github.com/larsartmann/go-cqrs-lite/kv/v4 => ../kv

replace github.com/larsartmann/go-cqrs-lite/listing/v4 => ../listing

replace github.com/larsartmann/go-cqrs-lite/otel/v4 => ../otel

replace github.com/larsartmann/go-cqrs-lite/query/v4 => ../query

replace github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot

replace github.com/larsartmann/go-cqrs-lite/scheduling/v4 => ../scheduling

replace github.com/larsartmann/go-cqrs-lite/schema/v4 => ../schema

replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../event/v4/eventtest
