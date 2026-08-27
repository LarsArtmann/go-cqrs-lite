module github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4

go 1.26.6

require (
	github.com/dgraph-io/dgo/v240 v240.2.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.12.0
	google.golang.org/grpc v1.83.0
)

require (
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.4.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.5.1 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260810153831-ec0a7760b754 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 => ../sqliteengine
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
	github.com/larsartmann/go-cqrs-lite/record/v4 => ../../record
)
