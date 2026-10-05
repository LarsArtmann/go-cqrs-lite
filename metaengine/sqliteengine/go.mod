module github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4

go 1.27

require (
	github.com/larsartmann/go-cqrs-lite/claiming/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.16.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.1
	github.com/onsi/ginkgo/v2 v2.33.0
	github.com/onsi/gomega v1.44.0
	modernc.org/sqlite v1.60.1
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/gkampitakis/go-snaps v0.5.23 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260926063103-aaccee046517 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.3 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.2 // indirect
	github.com/larsartmann/go-error-family v0.11.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.2 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

// Sibling replace for unpublished metaengine symbols (DurabilityReporter and later); stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../

replace github.com/larsartmann/go-cqrs-lite/claiming/v4 => ../../claiming
