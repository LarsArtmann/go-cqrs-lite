// Operator-config demo: load the deployment from a koanf YAML file and boot a
// system from it. This is the deployment-time story — the developer's domain
// code (DomainConfig) stays in Go; the operator's engine choices live in
// cqrs.yaml and can change per environment without a rebuild.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers the "sqlite" driver
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func runConfigFileDemo(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "quickstart-config")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "cqrs.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfig(dir)), 0o600); err != nil {
		return err
	}

	deployment, err := system.LoadConfig(configPath)
	if err != nil {
		return err
	}

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.Count("event_totals").On("demo.happened", struct{ ID string }{}, 1, "ID").Done(),
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		return err
	}

	defer sys.Close()

	if err := sys.Start(ctx); err != nil {
		return err
	}

	fmt.Println("config-file demo: system booted from", configPath)

	return nil
}

// sampleConfig writes the same shape an operator would keep under /etc: two
// engines (durable events, in-memory projections) and the two canonical
// instances. CQRS_ env overrides win over file values, e.g.
// CQRS_ENGINES__PRIMARY__DSN=/other/path/events.db.
func sampleConfig(dir string) string {
	return `engines:
  primary:
    driver: sqlite
    dsn: ` + filepath.Join(dir, "events.db") + `
    pragmas: ["journal_mode=WAL", "busy_timeout=5000"]
  projections:
    driver: memory
instances:
  - role: source-of-truth
    engine: primary
  - role: projections
    engine: projections
`
}
