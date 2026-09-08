module github.com/larsartmann/go-cqrs-lite/id/v4

go 1.26.5

require (
	github.com/larsartmann/go-branded-id v0.5.1
	github.com/larsartmann/go-error-family v0.10.0
	github.com/oklog/ulid/v2 v2.1.2
)

require (
	github.com/onsi/gomega v1.42.1
	pgregory.net/rapid v1.3.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/command/v4 => ../command

replace github.com/larsartmann/go-cqrs-lite/decider/v4 => ../decider

replace github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher

replace github.com/larsartmann/go-cqrs-lite/event/v4 => ../event

replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../event/v4/eventtest

replace github.com/larsartmann/go-cqrs-lite/id/v4 => ../id

replace github.com/larsartmann/go-cqrs-lite/kv/v4 => ../kv

replace github.com/larsartmann/go-cqrs-lite/metadata/v4 => ../metadata

replace github.com/larsartmann/go-cqrs-lite/query/v4 => ../query

replace github.com/larsartmann/go-cqrs-lite/record/v4 => ../record

replace github.com/larsartmann/go-cqrs-lite/schema/v4 => ../schema

replace github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot

replace github.com/larsartmann/go-cqrs-lite/storage/backuptest/v4 => ../storage/backuptest

replace github.com/larsartmann/go-cqrs-lite/storage/bbolt/v4 => ../storage/bbolt

replace github.com/larsartmann/go-cqrs-lite/storage/memory/v4 => ../storage/memory
