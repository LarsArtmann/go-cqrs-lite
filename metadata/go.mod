module github.com/larsartmann/go-cqrs-lite/metadata/v3

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/id/v3 v3.7.4

require (
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-error-family v0.6.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/id/v3 => ../id
