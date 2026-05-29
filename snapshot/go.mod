module github.com/larsartmann/go-cqrs-lite/snapshot

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/codec v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testhelpers v1.6.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec => ../codec
	github.com/larsartmann/go-cqrs-lite/event => ../event
	github.com/larsartmann/go-cqrs-lite/id => ../id
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
