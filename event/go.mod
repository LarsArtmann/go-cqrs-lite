module github.com/larsartmann/go-cqrs-lite/event

go 1.26.3

require (
	github.com/larsartmann/go-branded-id v0.3.0
	github.com/larsartmann/go-cqrs-lite/codec v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/command v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/memory v1.6.0
	github.com/larsartmann/go-cqrs-lite/query v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/schema v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testhelpers v1.6.0
	github.com/larsartmann/go-error-family v0.2.0
	github.com/oklog/ulid/v2 v2.1.1
	github.com/onsi/ginkgo/v2 v2.29.0
	github.com/onsi/gomega v1.41.0
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec => ../codec
	github.com/larsartmann/go-cqrs-lite/command => ../command
	github.com/larsartmann/go-cqrs-lite/id => ../id
	github.com/larsartmann/go-cqrs-lite/memory => ../memory
	github.com/larsartmann/go-cqrs-lite/query => ../query
	github.com/larsartmann/go-cqrs-lite/schema => ../schema
	github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers
)
