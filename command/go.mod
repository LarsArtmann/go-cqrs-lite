module github.com/larsartmann/go-cqrs-lite/command

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/dispatcher v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/testhelpers v1.6.0
	github.com/larsartmann/go-error-family v0.2.0
	github.com/onsi/ginkgo/v2 v2.29.0
	github.com/onsi/gomega v1.41.0
	github.com/samber/ro v0.3.0
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/larsartmann/go-branded-id v0.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/core v1.6.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/samber/lo v1.52.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260527015227-08cc5374adb3 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/dispatcher => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/id => ../id
)
