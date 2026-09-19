module github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4

go 1.27.1

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.14.0
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.5.0
)

require (
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/claiming/v4 v4.0.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.2 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.1 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.5.1 // indirect
	github.com/larsartmann/go-error-family v0.10.1 // indirect
	github.com/larsartmann/go-sse v0.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.48.0 // indirect
	modernc.org/libc v1.76.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.59.0 // indirect
)

// Sibling replaces for not-yet-tagged workspace modules; stripped by
// scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/scheduling/v4 => ../../scheduling

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../metaengine

replace github.com/larsartmann/go-cqrs-lite/claiming/v4 => ../../claiming

replace github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ../../metaengine/sqliteengine
