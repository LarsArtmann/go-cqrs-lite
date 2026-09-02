module github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4

go 1.26.7

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.12.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.4.0
	github.com/onsi/gomega v1.42.1
	go.etcd.io/bbolt v1.5.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.1 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.5.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.6.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// Sibling replace for unpublished metaengine symbols (DurabilityReporter and later); stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
