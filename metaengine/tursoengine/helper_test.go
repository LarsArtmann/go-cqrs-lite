package tursoengine_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewTursoEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := tursoengine.New("")
	if err != nil {
		tb.Skipf("turso not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func TestTursoEngineProfile(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewTursoEngine(t)

	profile := eng.Profile()
	g.Expect(profile.Name).To(gomega.Equal("sqlite")) // delegates to sqliteengine
	g.Expect(profile.Supports).To(gomega.HaveKey(metaengine.ADTMap))
}

func TestTursoDriverRegistered(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	drivers := metaengine.RegisteredDrivers()
	g.Expect(drivers).To(gomega.ContainElement("turso"))
}
