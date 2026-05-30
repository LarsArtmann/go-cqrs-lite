module github.com/larsartmann/go-cqrs-lite/example/listing

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/event v1.7.1
	github.com/larsartmann/go-cqrs-lite/id v1.7.1
	github.com/larsartmann/go-cqrs-lite/listing v1.7.1
	github.com/larsartmann/go-cqrs-lite/memory v1.7.1
)

require (
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot v1.7.1 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec => ../../codec
	github.com/larsartmann/go-cqrs-lite/dispatcher => ../../dispatcher
	github.com/larsartmann/go-cqrs-lite/event => ../../event
	github.com/larsartmann/go-cqrs-lite/id => ../../id
	github.com/larsartmann/go-cqrs-lite/listing => ../../listing
	github.com/larsartmann/go-cqrs-lite/memory => ../../memory
	github.com/larsartmann/go-cqrs-lite/snapshot => ../../snapshot
)
