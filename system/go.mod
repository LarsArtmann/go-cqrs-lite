module github.com/larsartmann/go-cqrs-lite/system/v4

go 1.26.5

require (
	github.com/knadh/koanf/parsers/yaml v1.1.1
	github.com/knadh/koanf/providers/env v1.1.0
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/v2 v2.3.6
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4 v4.0.0
	github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4 v4.0.0
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.8.0
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.1.0
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/watermill/v4 v4.3.0
	github.com/maypok86/otter/v2 v2.3.0
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/ThreeDotsLabs/watermill v1.5.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.14.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.5 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgraph-io/badger/v4 v4.9.6 // indirect
	github.com/dgraph-io/ristretto/v2 v2.4.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/getsentry/sentry-go v0.48.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/knadh/koanf/maps v0.1.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4 v4.0.0-20260808223431-099e6e1268fb // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.45.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp v0.0.0-20260727155853-b88d891fe743 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.75.3 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4 => ../commandlifecycle

replace github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4 => ../commandlifecycle/projections
