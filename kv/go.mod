module github.com/larsartmann/go-cqrs-lite/kv/v4

go 1.26.6

require (
	github.com/larsartmann/go-codec v0.2.0
	github.com/larsartmann/go-error-family v0.10.0
	github.com/maypok86/otter/v2 v2.3.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.3 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.58.0 // indirect
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
