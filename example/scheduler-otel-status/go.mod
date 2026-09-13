module github.com/larsartmann/go-cqrs-lite/example/scheduler-otel-status

go 1.26.7

require (
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/prometheus/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4 v4.4.0
	github.com/larsartmann/go-cqrs-lite/scheduling/v4 v4.4.0
	modernc.org/sqlite v1.58.0
)

// Examples pin published tags and run against the workspace in dev.
replace github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4 => ../../scheduling/sqlstore
