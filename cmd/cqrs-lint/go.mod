module github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint

go 1.26.4

require (
	github.com/larsartmann/go-finding v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-finding/pipeline v0.0.0-00010101000000-000000000000
	golang.org/x/tools v0.48.0
)

require (
	github.com/LarsArtmann/gogenfilter/v3 v3.3.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-faster/jx v1.2.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace (
	github.com/larsartmann/go-finding => ../../../go-finding
	github.com/larsartmann/go-finding/pipeline => ../../../go-finding/pipeline
)
