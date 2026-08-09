module github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4

go 1.26.5

require (
	github.com/dgraph-io/dgo/v240 v240.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.7.0
	google.golang.org/grpc v1.83.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.0.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260807164820-c8921c73eeea // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ../sqliteengine
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
	github.com/larsartmann/go-cqrs-lite/record/v4 => ../../record
)
