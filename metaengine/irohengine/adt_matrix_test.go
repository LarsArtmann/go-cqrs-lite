package irohengine_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestIrohADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "iroh",
			Create: func(t *testing.T) metaengine.Engine {
				return irohengine.Replicated(metaengine.NewMemoryEngine())
			},
		},
	})
}

func TestProfileIsReplicated(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := irohengine.Replicated(metaengine.NewMemoryEngine())
	defer eng.Close()

	p := eng.Profile()
	g.Expect(p.IsReplicated()).To(gomega.BeTrue())
	g.Expect(p.Replication).To(gomega.Equal(metaengine.ReplicationLeaderless))
	g.Expect(p.Name).To(gomega.ContainSubstring("iroh("))
}
