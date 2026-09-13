module github.com/larsartmann/go-cqrs-lite/example/scheduler-otel-status

go 1.26.7

require (
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/prometheus/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.4.0
	modernc.org/sqlite v1.58.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/claiming/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.66.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

// Examples pin published tags and run against the workspace in dev.
// sqlstore's StartedAt is unpublished, and claiming has no tag at all —
// both resolve to sibling checkouts (tag-release.sh strips these at cut).
replace github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4 => ../../scheduling/sqlstore

replace github.com/larsartmann/go-cqrs-lite/claiming/v4 => ../../claiming
