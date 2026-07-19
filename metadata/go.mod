module github.com/larsartmann/go-cqrs-lite/metadata/v4

go 1.26.4

replace github.com/larsartmann/go-cqrs-lite/id/v4 => ../id

require github.com/larsartmann/go-cqrs-lite/id/v4 v4.0.0-00010101000000-000000000000

require (
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-error-family v0.7.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)
