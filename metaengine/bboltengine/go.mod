module github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4

go 1.27

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.17.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2
	github.com/onsi/gomega v1.44.0
	go.etcd.io/bbolt v1.5.0
)

require (
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.4 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.2 // indirect
	github.com/larsartmann/go-error-family v0.11.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

// Sibling replace for unpublished metaengine symbols (DurabilityReporter and later); stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
