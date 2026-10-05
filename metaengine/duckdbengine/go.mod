module github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4

go 1.27

require (
	github.com/duckdb/duckdb-go/v2 v2.10506.0
	github.com/larsartmann/go-cqrs-lite/claiming/v4 v4.0.1
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.16.0
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.1
)

require (
	github.com/andybalholm/brotli v1.2.6 // indirect
	github.com/apache/arrow-go/v18 v18.8.0 // indirect
	github.com/duckdb/duckdb-go-bindings v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-arm64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-amd64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-arm64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/windows-amd64 v0.10506.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.11.2 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.20.1 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.3 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.2 // indirect
	github.com/larsartmann/go-error-family v0.11.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.33 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/sys v0.48.0 // indirect
)

// Sibling replace for unpublished metaengine symbols (planned-table capabilities); stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../

replace github.com/larsartmann/go-cqrs-lite/claiming/v4 => ../../claiming
