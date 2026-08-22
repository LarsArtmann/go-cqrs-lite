module github.com/larsartmann/go-cqrs-lite/query/v4

go 1.26.6

require (
	github.com/larsartmann/go-codec v0.2.0
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.5.0
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/storage/memory/v4 v4.2.0
	github.com/larsartmann/go-error-family v0.10.0
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.1
	pgregory.net/rapid v1.3.0
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

// v4.6.0 pinned metadata/v4.4.0 while using metadata.Metadata (v4.5.0): unbuildable for consumers.
retract v4.6.0
