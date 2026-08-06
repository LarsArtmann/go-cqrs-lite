package bench

import (
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	mysql "github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// BenchmarkBenchkitSuite_MySQL runs the full benchkit suite against MySQL/MariaDB.
// Requires a live server. Set MYSQL_BENCH_DSN to enable:
//
//	MYSQL_BENCH_DSN="root:pass@tcp(localhost:3306)/bench?parseTime=true" \
//	    go test -bench=BenchmarkBenchkitSuite_MySQL -benchtime=1x ./stack/bench/
//
// Example with Docker:
//
//	docker run --rm -p 3306:3306 -e MYSQL_ROOT_PASSWORD=pass -d mariadb:11
//
// Or with Nix:
//
//	nix run .#integration-mysql-nspawn
func BenchmarkBenchkitSuite_MySQL(b *testing.B) {
	dsn := os.Getenv("MYSQL_BENCH_DSN")
	if dsn == "" {
		b.Skip("MYSQL_BENCH_DSN not set — skipping MySQL benchmark. " +
			"Set MYSQL_BENCH_DSN to a live MySQL/MariaDB DSN to enable.")
	}

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "mysql",
	}, func() (*stack.Bundle, error) {
		return mysql.New(dsn)
	})
}
