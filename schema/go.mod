module github.com/larsartmann/go-cqrs-lite/schema/v2

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.1.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.1.0
	github.com/larsartmann/go-cqrs-lite/memory/v2 v2.1.0
)

require (
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 v2.1.0 // indirect
	github.com/larsartmann/go-error-family v0.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../codec
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../id
	github.com/larsartmann/go-cqrs-lite/memory/v2 => ../memory
	github.com/larsartmann/go-cqrs-lite/snapshot/v2 => ../snapshot
)
