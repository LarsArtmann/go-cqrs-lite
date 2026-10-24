package metaengine_test

import (
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type probeResultV1 struct {
	ID       string
	Status   string
	Priority string
}

type probeResultV2 struct {
	ID       string
	Status   string
	Priority int
}

func TestStampTypeDriftProbe(t *testing.T) {
	eng := openStampEngine(t)

	q1 := metaengine.Query[stampQueryInput, probeResultV1]("stamp_tasks",
		metaengine.OnRecord(stampEvent{}, func(_ record.Record, e stampEvent) (string, probeResultV1) {
			return e.ID, probeResultV1{ID: e.ID}
		}),
		metaengine.FilterOnField[probeResultV1]("status", metaengine.FilterEq),
		metaengine.SortOnField[probeResultV1]("priority", true),
	)
	s1, err := metaengine.Plan([]metaengine.Engine{eng}, q1)
	fmt.Printf("boot1: err=%v\n", err)
	if s1 != nil {
		t.Cleanup(func() { _ = s1.Close() })
		for _, lp := range s1.Plan().LayoutPlans {
			fmt.Printf("boot1 plan cols=%v fp=%s\n", lp.ColumnNames(), lp.Fingerprint())
		}
	}

	q2 := metaengine.Query[stampQueryInput, probeResultV2]("stamp_tasks",
		metaengine.OnRecord(stampEvent{}, func(_ record.Record, e stampEvent) (string, probeResultV2) {
			return e.ID, probeResultV2{ID: e.ID}
		}),
		metaengine.FilterOnField[probeResultV2]("status", metaengine.FilterEq),
		metaengine.SortOnField[probeResultV2]("priority", true),
	)
	s2, err := metaengine.Plan([]metaengine.Engine{eng}, q2)
	fmt.Printf("boot2: err=%v\n", err)
	if s2 != nil {
		t.Cleanup(func() { _ = s2.Close() })
	}
}
