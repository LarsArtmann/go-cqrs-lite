module github.com/larsartmann/go-cqrs-lite/integration/v4

go 1.26.4

require (
	github.com/cockroachdb/pebble v1.1.5
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/encryption/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/graph/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/idempotency/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/middleware/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/signing/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/stack/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/pebble/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testutil/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/watermill/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-error-family v0.7.0
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.1
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/sdk/metric v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
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
	github.com/getsentry/sentry-go v0.48.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260604005048-7023385849c0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.0.0-00010101000000-000000000000 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mattn/go-isatty v0.0.23 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.0 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/exp v0.0.0-20260709172345-9ea1abe57597 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/sqlite v1.54.0 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v4 => ../command
	github.com/larsartmann/go-cqrs-lite/decider/v4 => ../decider
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/encryption/v4 => ../encryption
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
	github.com/larsartmann/go-cqrs-lite/idempotency/v4 => ../idempotency
	github.com/larsartmann/go-cqrs-lite/kv/v4 => ../kv
	github.com/larsartmann/go-cqrs-lite/listing/v4 => ../listing
	github.com/larsartmann/go-cqrs-lite/middleware/v4 => ../middleware
	github.com/larsartmann/go-cqrs-lite/otel/v4 => ../otel
	github.com/larsartmann/go-cqrs-lite/query/v4 => ../query
	github.com/larsartmann/go-cqrs-lite/signing/v4 => ../signing
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../storage/memory
	github.com/larsartmann/go-cqrs-lite/storage/pebble/v4 => ../storage/pebble
	github.com/larsartmann/go-cqrs-lite/storage/v4 => ../storage
	github.com/larsartmann/go-cqrs-lite/testutil/v4 => ../testutil
)

replace github.com/larsartmann/go-cqrs-lite/graph/v4 => ../graph

replace github.com/larsartmann/go-cqrs-lite/projection/v4 => ../projection

replace github.com/larsartmann/go-cqrs-lite/stack/v4 => ../stack

replace github.com/larsartmann/go-cqrs-lite/watermill/v4 => ../watermill

replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../event/v4/eventtest

replace github.com/larsartmann/go-cqrs-lite/dedup/v4 => ../dedup

replace github.com/larsartmann/go-cqrs-lite/scheduling/v4 => ../scheduling

replace github.com/larsartmann/go-cqrs-lite/metadata/v4 => ../metadata
