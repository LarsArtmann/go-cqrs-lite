package systemscenario_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// BenchmarkScenarioBoot measures the full per-scenario cost a harness test
// pays: system.New + middleware + Start + one given + one act + one Then +
// GracefulClose. The pilot suite's suite-time delta claim (recipes §2.43)
// cites this number.
func BenchmarkScenarioBoot(b *testing.B) {
	streamID := id.NewStreamID()
	ctx := context.Background()

	for b.Loop() {
		sc := systemscenario.System(b, ctx, taskDomain(), memoryDeployment())
		sc.Given().Command(newTaskCmd("task.create", streamID)).
			When(newTaskCmd("task.rename", streamID)).
			Then("task.updated")
	}
}
