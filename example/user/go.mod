module github.com/larsartmann/go-cqrs-lite/example/user

go 1.26.2

require (
	github.com/larsartmann/go-cqrs-lite/catalog v0.0.0
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
	github.com/larsartmann/go-cqrs-lite/memory v0.0.0
)

require (
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/getsentry/sentry-go v0.45.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/go-faster/jx v1.0.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-json-experiment/json v0.0.0-20260430182902-b6187a392ed4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/catalog => ../../catalog
	github.com/larsartmann/go-cqrs-lite/core => ../../core
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
)
