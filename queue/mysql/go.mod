module github.com/larsartmann/go-cqrs-lite/queue/mysql/v4

go 1.26.7

require (
	github.com/go-sql-driver/mysql v1.10.1
	github.com/larsartmann/go-cqrs-lite/queue/v4 v4.0.0
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/larsartmann/go-error-family v0.10.1 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/queue/v4 => ../../queue
