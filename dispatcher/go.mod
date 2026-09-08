module github.com/larsartmann/go-cqrs-lite/dispatcher/v4

go 1.26.5

require github.com/larsartmann/go-error-family v0.10.0

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
