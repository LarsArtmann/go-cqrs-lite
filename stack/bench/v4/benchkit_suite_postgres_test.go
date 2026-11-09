package bench

import (
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// BenchmarkBenchkitSuite_Postgres runs the full benchkit suite against PostgreSQL.
// Requires a live server. Set POSTGRES_BENCH_DSN to enable:
//
//	POSTGRES_BENCH_DSN="postgres://user:pass@localhost:5432/bench?sslmode=disable" \
//	    go test -bench=BenchmarkBenchkitSuite_Postgres -benchtime=1x ./stack/bench/
//
// Example with Docker:
//
//	docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=bench -d postgres:16-alpine
//
// Or with Nix:
//
//	nix run .#integration-pg
func BenchmarkBenchkitSuite_Postgres(b *testing.B) {
	dsn := os.Getenv("POSTGRES_BENCH_DSN")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_TEST_DSN")
	}

	if dsn == "" {
		b.Skip("POSTGRES_BENCH_DSN (or POSTGRES_TEST_DSN) not set — skipping Postgres benchmark. " +
			"Set the DSN to a live PostgreSQL server to enable.")
	}

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "postgres",
	}, func() (*stack.Bundle, error) {
		return postgres.New(dsn)
	})
}
