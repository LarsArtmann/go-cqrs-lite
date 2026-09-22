module github.com/larsartmann/go-cqrs-lite/storage/turso/v4

go 1.27.1

require (
	github.com/gkampitakis/go-snaps v0.5.23
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.11.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.11.1
	github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.4.0
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.6.1
	github.com/larsartmann/go-cqrs-lite/kv/v4 v4.3.1
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.5.0
	github.com/larsartmann/go-cqrs-lite/query/v4 v4.8.1
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.0
	github.com/larsartmann/go-cqrs-lite/snapshot/v4 v4.5.1
	github.com/larsartmann/go-cqrs-lite/storage/v4 v4.10.1
	github.com/larsartmann/go-error-family v0.10.1
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/trace v1.46.0
	turso.tech/database/tursogo v0.7.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/gkampitakis/ciinfo v0.3.4 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.6.0 // indirect
	github.com/larsartmann/go-codec v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.4.1 // indirect
	github.com/larsartmann/go-cqrs-lite/listing/v4 v4.4.1 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.4.0 // indirect
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.5.0 // indirect
	github.com/larsartmann/go-sqlitestore v0.1.0 // indirect
	github.com/maruel/natural v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mattn/go-sqlite3 v1.14.52 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tursodatabase/turso-go-platform-libs v0.7.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.46.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.48.0 // indirect
	modernc.org/libc v1.76.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.59.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/record/v4 => ../../record
