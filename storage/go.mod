module github.com/larsartmann/go-cqrs-lite/storage

go 1.26.2

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/larsartmann/go-cqrs-lite/core v0.0.0
)

require (
	github.com/getsentry/sentry-go v0.45.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/larsartmann/go-branded-id v0.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/core => ../core
