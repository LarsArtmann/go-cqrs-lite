// goal-shaped-app — "The Goal in 5 minutes" for go-cqrs-lite.
//
// Developers declare types (domain.go) and rules (app.go). Operators decide
// where data lives (cqrs.yaml, plus CQRS_ env overrides): swap sqlite to
// postgres by changing one driver line — both drivers are compiled in, the
// swap is a config change, not a code change. After the story runs, main
// prints EXPLAIN and Doctor so the planner explains every placement.
//
// Run: GOWORK=off go run .            (uses ./cqrs.yaml, sqlite at goal.db)
//
//	Swap: CQRS_ENGINES__PRIMARY__DRIVER=postgres \
//	      CQRS_ENGINES__PRIMARY__DSN=postgres://user:pass@host/goal go run .
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"     // registers the "postgres" driver
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers the "sqlite" driver
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func main() {
	configPath := flag.String("config", "cqrs.yaml", "operator deployment config")

	flag.Parse()

	if err := run(context.Background(), *configPath); err != nil {
		log.Fatal(err)
	}
}

// run is the whole operator surface: load the deployment, compose, run the
// story, explain the plan. The developer-owned Domain() is identical for
// every deployment.
func run(ctx context.Context, configPath string) error {
	deployment, err := system.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load operator config %q: %w", configPath, err)
	}

	sys, err := system.New(ctx, Domain(), deployment)
	if err != nil {
		return fmt.Errorf("compose system: %w", err)
	}
	defer metaengine.DeferClose(sys)

	view, err := runStory(ctx, sys)
	if err != nil {
		return err
	}

	fmt.Printf("\nTaskView: %+v\n", view)
	fmt.Println()

	fmt.Println(sys.MetaEngine().ExplainPlan())
	fmt.Println(sys.MetaEngine().Doctor(ctx))

	return nil
}
