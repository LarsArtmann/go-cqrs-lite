module github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4

go 1.27

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.17.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2
	github.com/onsi/gomega v1.44.0
	go.etcd.io/bbolt v1.5.0
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260926063103-aaccee046517 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.4 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.2 // indirect
	github.com/larsartmann/go-error-family v0.11.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/onsi/ginkgo/v2 v2.33.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
	modernc.org/sqlite v1.60.1 // indirect
)

// Sibling replace for unpublished metaengine symbols (DurabilityReporter and later); stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
