module github.com/larsartmann/go-cqrs-lite/catalog

go 1.26.2

require (
	github.com/go-faster/yaml v0.4.6
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
)

require (
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/go-faster/jx v1.0.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/sys v0.43.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/core => ../core
