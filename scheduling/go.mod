module github.com/larsartmann/go-cqrs-lite/scheduling/v4

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/testutil/v4 v4.0.0

require pgregory.net/rapid v1.3.0 // indirect
replace github.com/larsartmann/go-cqrs-lite/id/v4 => ../id
replace github.com/larsartmann/go-cqrs-lite/testutil/v4 => ../testutil
replace github.com/larsartmann/go-cqrs-lite/codec/v4 => ../codec
replace github.com/larsartmann/go-cqrs-lite/command/v4 => ../command
replace github.com/larsartmann/go-cqrs-lite/metadata/v4 => ../metadata
replace github.com/larsartmann/go-cqrs-lite/dispatcher/v4 => ../dispatcher
replace github.com/larsartmann/go-cqrs-lite/event/v4 => ../event
replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../event/v4/eventtest
replace github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot

