module github.com/larsartmann/go-cqrs-lite/storage

go 1.26.2

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
	modernc.org/sqlite v1.50.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.9.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/tursodatabase/turso-go-platform-libs v0.5.3 // indirect
	golang.org/x/sys v0.43.0 // indirect
	modernc.org/libc v1.72.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	turso.tech/database/tursogo v0.5.3 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/core => ../core
