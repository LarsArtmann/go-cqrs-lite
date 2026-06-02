module github.com/larsartmann/go-cqrs-lite/integration

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/command v1.7.1
	github.com/larsartmann/go-cqrs-lite/decider v1.7.1
	github.com/larsartmann/go-cqrs-lite/dispatcher v1.7.1
	github.com/larsartmann/go-cqrs-lite/event v1.7.1
	github.com/larsartmann/go-cqrs-lite/id v1.7.1
	github.com/larsartmann/go-cqrs-lite/memory v1.7.1
	github.com/larsartmann/go-cqrs-lite/middleware v1.7.1
	github.com/larsartmann/go-cqrs-lite/otel v1.7.1
	github.com/larsartmann/go-cqrs-lite/projection v1.7.1
	github.com/larsartmann/go-cqrs-lite/query v1.7.1
	github.com/larsartmann/go-cqrs-lite/signing v1.7.1
	github.com/larsartmann/go-cqrs-lite/storage v1.7.1
	github.com/onsi/ginkgo/v2 v2.29.0
	github.com/onsi/gomega v1.41.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/codec v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/listing v1.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/snapshot v1.7.1 // indirect
	github.com/larsartmann/go-error-family v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/command => ../command
	github.com/larsartmann/go-cqrs-lite/decider => ../decider
	github.com/larsartmann/go-cqrs-lite/dispatcher => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event => ../event
	github.com/larsartmann/go-cqrs-lite/id => ../id
	github.com/larsartmann/go-cqrs-lite/memory => ../memory
	github.com/larsartmann/go-cqrs-lite/middleware => ../middleware
	github.com/larsartmann/go-cqrs-lite/otel => ../otel
	github.com/larsartmann/go-cqrs-lite/projection => ../projection
	github.com/larsartmann/go-cqrs-lite/query => ../query
	github.com/larsartmann/go-cqrs-lite/signing => ../signing
	github.com/larsartmann/go-cqrs-lite/storage => ../storage
)
