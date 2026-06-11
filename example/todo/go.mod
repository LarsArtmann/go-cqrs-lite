module github.com/larsartmann/go-cqrs-lite/example/todo

go 1.26.3

require (
	github.com/cockroachdb/pebble v1.1.5
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/decider/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/memory/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/pebble/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/projection/v2 v2.2.0
	github.com/larsartmann/go-cqrs-lite/query/v2 v2.2.0
	github.com/larsartmann/httputil v0.1.1
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.13.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/getsentry/sentry-go v0.46.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/otel/v2 v2.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 v2.2.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.68.1 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/exp v0.0.0-20260603202125-055de637280b // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../../codec
	github.com/larsartmann/go-cqrs-lite/command/v2 => ../../command
	github.com/larsartmann/go-cqrs-lite/decider/v2 => ../../decider
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../../event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../../id
	github.com/larsartmann/go-cqrs-lite/memory/v2 => ../../memory
	github.com/larsartmann/go-cqrs-lite/otel/v2 => ../../otel
	github.com/larsartmann/go-cqrs-lite/pebble/v2 => ../../pebble
	github.com/larsartmann/go-cqrs-lite/projection/v2 => ../../projection
	github.com/larsartmann/go-cqrs-lite/query/v2 => ../../query
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../../snapshot
)
