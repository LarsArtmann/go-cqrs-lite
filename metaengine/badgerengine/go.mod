module github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4

go 1.26.7

require (
	github.com/dgraph-io/badger/v4 v4.9.6
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.12.0
	github.com/onsi/gomega v1.42.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.4.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.1 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.4.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.6.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

// Sibling replace for unpublished metaengine symbols (DurabilityReporter and later); stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
