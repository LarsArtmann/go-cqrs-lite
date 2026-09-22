module github.com/larsartmann/go-cqrs-lite/metaengine/otelobserver/v4

go 1.27.1

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.14.0
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.5.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.0
	github.com/larsartmann/go-error-family v0.10.1
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/sdk/metric v1.46.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.2 // indirect
	github.com/larsartmann/go-sse v0.6.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

// Sibling replace for unpublished metaengine symbols (health hooks:
// OnQuarantined/OnReactivated/OnProbe/OnCatchUp, Hooks.Merge, CurrentHooks);
// stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
