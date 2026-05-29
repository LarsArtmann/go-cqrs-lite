module github.com/larsartmann/go-cqrs-lite/testhelpers

go 1.26.3

require github.com/larsartmann/go-cqrs-lite/event v1.7.1

require (
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/command v1.7.1
	github.com/larsartmann/go-cqrs-lite/id v1.7.1
	github.com/larsartmann/go-cqrs-lite/query v1.7.1
	github.com/larsartmann/go-cqrs-lite/snapshot v1.7.1
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
)
replace (
	github.com/larsartmann/go-cqrs-lite/codec => ../codec
	github.com/larsartmann/go-cqrs-lite/command => ../command
	github.com/larsartmann/go-cqrs-lite/event => ../event
	github.com/larsartmann/go-cqrs-lite/id => ../id
	github.com/larsartmann/go-cqrs-lite/query => ../query
	github.com/larsartmann/go-cqrs-lite/snapshot => ../snapshot
)
