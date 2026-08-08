module github.com/larsartmann/go-cqrs-lite/example/taskmanager

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/idempotency/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/middleware/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/scenario/v4 v4.1.0
	github.com/larsartmann/go-cqrs-lite/signing/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/system/v4 v4.1.0
	github.com/larsartmann/go-cqrs-lite/transport/http/v4 v4.2.0
	github.com/larsartmann/go-error-family v0.10.0
	go.opentelemetry.io/otel v1.45.0
)

require (
	github.com/bits-and-blooms/bitset v1.24.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/failsafe-go/failsafe-go v0.9.6 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/parsers/yaml v1.1.1 // indirect
	github.com/knadh/koanf/providers/env v1.1.0 // indirect
	github.com/knadh/koanf/providers/file v1.2.1 // indirect
	github.com/knadh/koanf/v2 v2.3.6 // indirect
	github.com/larsartmann/go-cqrs-lite/flightrecorder/v4 v4.0.0-20260808102531-55cd247546d5 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.0.0 // indirect
	github.com/larsartmann/go-idempotency v0.1.2 // indirect
	github.com/larsartmann/go-retry v0.3.1 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.5.0 // indirect
	github.com/larsartmann/go-must v0.1.2
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.45.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	modernc.org/libc v1.75.2 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.0 // indirect
	modernc.org/sqlite v1.56.0 // indirect
)

replace github.com/larsartmann/go-must => /home/lars/projects/go-must
