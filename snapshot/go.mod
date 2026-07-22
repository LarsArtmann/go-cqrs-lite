module github.com/larsartmann/go-cqrs-lite/snapshot/v4

go 1.26.4

require github.com/larsartmann/go-error-family v0.7.0
replace (
	github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
)

