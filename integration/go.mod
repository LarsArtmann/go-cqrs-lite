module github.com/larsartmann/go-cqrs-lite/integration

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/core v1.6.0
	github.com/larsartmann/go-cqrs-lite/memory v1.6.0
	github.com/larsartmann/go-cqrs-lite/middleware v1.6.0
	github.com/larsartmann/go-cqrs-lite/projection v1.6.0
	github.com/larsartmann/go-cqrs-lite/signing v1.6.0
	github.com/larsartmann/go-cqrs-lite/storage v1.6.0
	github.com/larsartmann/go-cqrs-lite/testhelpers v1.6.0
	github.com/onsi/ginkgo/v2 v2.29.0
	github.com/onsi/gomega v1.41.0
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.13.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.5 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/getsentry/sentry-go v0.46.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/saga v1.6.0 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/tursodatabase/turso-go-platform-libs v0.6.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260508232706-74f9aab9d74a // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	turso.tech/database/tursogo v0.6.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/core => ../core
	github.com/larsartmann/go-cqrs-lite/memory => ../memory
	github.com/larsartmann/go-cqrs-lite/middleware => ../middleware
	github.com/larsartmann/go-cqrs-lite/projection => ../projection
	github.com/larsartmann/go-cqrs-lite/saga => ../saga
	github.com/larsartmann/go-cqrs-lite/signing => ../signing
	github.com/larsartmann/go-cqrs-lite/storage => ../storage
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
