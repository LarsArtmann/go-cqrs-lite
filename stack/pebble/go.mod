module github.com/larsartmann/go-cqrs-lite/stack/pebble/v3

go 1.26.3

require (
	github.com/cockroachdb/pebble v1.1.5
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/kv/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/stack/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/storage/pebble/v3 v3.0.0
	github.com/larsartmann/go-cqrs-lite/watermill/v3 v3.0.0
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
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/getsentry/sentry-go v0.47.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/decider/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/query/v3 v3.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v3 v3.0.0 // indirect
	github.com/larsartmann/go-error-family v0.5.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.69.0 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v3 => ../../codec
	github.com/larsartmann/go-cqrs-lite/command/v3 => ../../command
	github.com/larsartmann/go-cqrs-lite/decider/v3 => ../../decider
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v3 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v3 => ../../id
	github.com/larsartmann/go-cqrs-lite/kv/v3 => ../../kv
	github.com/larsartmann/go-cqrs-lite/otel/v3 => ../../otel
	github.com/larsartmann/go-cqrs-lite/query/v3 => ../../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v3 => ../../snapshot
	github.com/larsartmann/go-cqrs-lite/stack/v3 => ..
	github.com/larsartmann/go-cqrs-lite/storage/memory/v3 => ../../storage/memory
	github.com/larsartmann/go-cqrs-lite/storage/pebble/v3 => ../../storage/pebble
)

replace github.com/larsartmann/go-cqrs-lite/watermill/v3 => ../../watermill
