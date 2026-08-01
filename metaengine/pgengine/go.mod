module github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4

go 1.26.5

require (
	github.com/jackc/pgx/v5 v5.9.2
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
