module github.com/larsartmann/go-cqrs-lite/metaengine/v4

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.0.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.0.0
	github.com/larsartmann/go-sse v0.4.0
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.1
	modernc.org/sqlite v1.56.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	modernc.org/libc v1.74.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1
	github.com/gkampitakis/go-snaps v0.5.23 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ./sqliteengine

replace github.com/larsartmann/go-cqrs-lite/record/v4 => ../record
