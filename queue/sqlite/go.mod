module github.com/larsartmann/go-cqrs-lite/queue/sqlite/v4

go 1.27.2

require (
	github.com/larsartmann/go-cqrs-lite/claiming/v4 v4.0.2
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.20.0
	github.com/larsartmann/go-cqrs-lite/queue/v4 v4.0.3
	modernc.org/sqlite v1.61.0
)

require (
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-codec v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-error-family v0.12.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.49.0 // indirect
	modernc.org/libc v1.79.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/queue/v4 => ../../queue

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../metaengine

replace github.com/larsartmann/go-cqrs-lite/claiming/v4 => ../../claiming
