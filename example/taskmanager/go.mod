module github.com/larsartmann/go-cqrs-lite/example/taskmanager

go 1.26.7

require (
	github.com/larsartmann/go-codec v0.2.0
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.8.1
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.5.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.9.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.5.0
	github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4 v4.4.1
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.12.0
	github.com/larsartmann/go-cqrs-lite/middleware/v4 v4.5.1
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/projectionhost/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.7.1
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/scenario/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/signing/v4 v4.2.1
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/system/v4 v4.6.0
	github.com/larsartmann/go-error-family v0.10.0
	github.com/larsartmann/go-idempotency v0.3.0
	go.opentelemetry.io/otel v1.46.0
)

require (
	github.com/ThreeDotsLabs/watermill v1.5.3 // indirect
	github.com/bits-and-blooms/bitset v1.25.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/failsafe-go/failsafe-go v0.9.7 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/getsentry/sentry-go v0.49.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/knadh/koanf/maps v0.1.3 // indirect
	github.com/knadh/koanf/parsers/yaml v1.1.1 // indirect
	github.com/knadh/koanf/providers/env v1.1.0 // indirect
	github.com/knadh/koanf/providers/file v1.2.1 // indirect
	github.com/knadh/koanf/v2 v2.3.6 // indirect
	github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4 v4.0.1 // indirect
	github.com/larsartmann/go-cqrs-lite/watermill/v4 v4.5.1 // indirect
	github.com/larsartmann/go-flightrecorder v0.2.0 // indirect
	github.com/larsartmann/go-retry v0.5.0 // indirect
	github.com/larsartmann/go-sse v0.6.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/moby/api v1.56.0 // indirect
	github.com/moby/moby/client v0.6.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.71.0 // indirect
	github.com/prometheus/procfs v0.22.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/shirou/gopsutil/v4 v4.26.8 // indirect
	github.com/sirupsen/logrus v1.10.2 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.71.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.56.0 // indirect
	golang.org/x/exp v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260904194346-d0f1323225a4 // indirect
	google.golang.org/grpc v1.83.2 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.58.0 // indirect
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.3 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.8.1 // indirect
	github.com/larsartmann/go-must v0.1.2
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)
