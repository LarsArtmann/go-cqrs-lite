module github.com/larsartmann/go-cqrs-lite/metaengine/bench/v4

go 1.26.5

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.0.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.6.0
	modernc.org/sqlite v1.56.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.0.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.47.0 // indirect
	modernc.org/libc v1.75.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4 => ../duckdbengine
	github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4 => ../pebbleengine
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ../sqliteengine
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
)
