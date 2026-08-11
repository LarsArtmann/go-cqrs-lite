package projections_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestDeclarations_ConstructWithoutPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func() any
	}{
		{"DeadLetterQueue", func() any { return projections.DeadLetterQueue() }},
		{"RetryCount", func() any { return projections.RetryCount() }},
		{"FailureLog", func() any { return projections.FailureLog() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(tt.fn()).NotTo(BeNil())
		})
	}
}

func TestAll_ReturnsThreeDeclarations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	all := projections.All()
	g.Expect(all).To(HaveLen(3))
}

func TestAll_ProjectionsPlanTogether(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		projections.All()...,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(store).NotTo(BeNil())
}

func TestRetryCount_AppliesAndCounts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		projections.RetryCount(),
	)
	g.Expect(err).NotTo(HaveOccurred())

	cmdID := id.NewCommandID()

	retryPayload := commandlifecycle.RetriedPayload{
		CommandType: "create_user",
		Attempt:     1,
	}

	for range 3 {
		g.Expect(store.ApplyRecord(
			context.Background(),
			makeRecord("command.retried", cmdID),
			retryPayload,
		)).To(Succeed())
	}

	result, err := metaengine.ExecuteTyped[projections.RetryCountQuery, map[string]int64](
		context.Background(), store, projections.RetryCountQuery{},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result[cmdID.String()]).To(Equal(int64(3)))
}

func TestDeadLetterQueue_AppliesAndStores(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		projections.DeadLetterQueue(),
	)
	g.Expect(err).NotTo(HaveOccurred())

	cmdID := id.NewCommandID()

	dlPayload := commandlifecycle.DeadLetteredPayload{
		CommandType: "create_user",
		Error:       "database timeout",
		Attempts:    3,
	}

	g.Expect(store.ApplyRecord(
		context.Background(),
		makeRecord("command.dead-lettered", cmdID),
		dlPayload,
	)).To(Succeed())
}

func TestFailureLog_AppliesAndAppends(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		projections.FailureLog(),
	)
	g.Expect(err).NotTo(HaveOccurred())

	for i := range 3 {
		g.Expect(store.ApplyRecord(
			context.Background(),
			makeRecord("command.failed", id.NewCommandID()),
			commandlifecycle.FailedPayload{
				CommandType: "create_user",
				Error:       "timeout",
				Attempt:     i + 1,
			},
		)).To(Succeed())
	}
}

func makeRecord(eventType string, cmdID id.CommandID) record.Record {
	causationID, _ := id.ParseCausationID(cmdID.String())

	return record.Record{
		Type:       eventType,
		StreamType: string(commandlifecycle.StreamTypeCommandLifecycle),
		StreamID:   record.NewStreamRef("CommandLifecycle", cmdID.String()),
		MetaData: record.CommonMetadata{
			CausationID: causationID,
		},
	}
}
