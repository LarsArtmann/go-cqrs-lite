package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// openPostgres benchmarks the PostgreSQL preset against the --dsn server.
func openPostgres(env openEnv) (binding, error) {
	var opts []postgres.Option
	if env.durabilitySet {
		opts = append(opts, postgres.WithDurability(env.durability))
	}

	return serverBinding(env, "postgres", func() (*stack.Bundle, error) {
		bundle, err := postgres.New(env.dsn, opts...)
		if err != nil {
			return nil, fmt.Errorf("open postgres store: %w", err)
		}

		return bundle, nil
	}), nil
}

// openMySQL benchmarks the MySQL/MariaDB preset against the --dsn server.
func openMySQL(env openEnv) (binding, error) {
	return serverBinding(env, "mysql", func() (*stack.Bundle, error) {
		bundle, err := mysql.New(env.dsn)
		if err != nil {
			return nil, fmt.Errorf("open mysql store: %w", err)
		}

		return bundle, nil
	}), nil
}

// serverBinding prepares a backend on a database server. The tool cannot
// give every run its own database there, so repeats, soak iterations, and
// the warmup store all share the one the DSN points at; the warning says so
// instead of letting the samples pass for independent.
func serverBinding(env openEnv, backend string, open func() (*stack.Bundle, error)) binding {
	bound := binding{name: backend, factory: open}

	if env.generations || env.calls.warmup {
		bound.warnings = append(bound.warnings, fmt.Sprintf(
			"backend %q reopens the same database for every run, so repeats, soak iterations, "+
				"and the warmup store are not isolated; start from an empty database to keep samples independent",
			backend,
		))
	}

	return bound
}
