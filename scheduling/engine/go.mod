module github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4

go 1.26.7

require (
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.13.0
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.4.0
)

require (
	github.com/larsartmann/go-codec v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/claiming/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.1 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.5.0 // indirect
	github.com/larsartmann/go-error-family v0.10.1 // indirect
	github.com/larsartmann/go-sse v0.6.0 // indirect
	github.com/onsi/ginkgo/v2 v2.32.1 // indirect
	github.com/onsi/gomega v1.43.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/sqlite v1.59.0 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

// Sibling replaces for not-yet-tagged workspace modules; stripped by
// scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/scheduling/v4 => ../../scheduling

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../metaengine

replace github.com/larsartmann/go-cqrs-lite/claiming/v4 => ../../claiming

replace github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ../../metaengine/sqliteengine
