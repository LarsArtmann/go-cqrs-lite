module github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/benchkit/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4 v4.0.0-20260731174855-00ddf243d19f
	github.com/larsartmann/go-cqrs-lite/stack/memory/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/stack/pebble/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/stack/postgres/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/stack/turso/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/stack/v4 v4.2.1-0.20260731174855-00ddf243d19f
	github.com/larsartmann/go-error-family v0.10.0
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/ThreeDotsLabs/watermill v1.5.2 // indirect
	github.com/apache/arrow-go/v18 v18.7.0 // indirect
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
	github.com/duckdb/duckdb-go-bindings v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-arm64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-amd64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-arm64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/windows-amd64 v0.10505.0 // indirect
	github.com/duckdb/duckdb-go/v2 v2.10505.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/getsentry/sentry-go v0.48.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/pebble/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/turso/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/watermill/v4 v4.2.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/exp v0.0.0-20260727155853-b88d891fe743 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.74.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.55.0 // indirect
)
