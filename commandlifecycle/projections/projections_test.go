package projections_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	. "github.com/onsi/gomega"
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
		{"ProcessingTime", func() any { return projections.ProcessingTime() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(tt.fn()).NotTo(BeNil())
		})
	}
}

func TestAll_ReturnsFourDeclarations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	all := projections.All()
	g.Expect(all).To(HaveLen(4))
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

	result, err := metaengine.ExecuteTyped[projections.DeadLetterQuery, projections.DeadLetterEntry](
		context.Background(),
		store,
		projections.DeadLetterQuery{CommandID: cmdID.String()},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.CommandType).To(Equal("create_user"))
	g.Expect(result.Error).To(Equal("database timeout"))
	g.Expect(result.Attempts).To(Equal(3))
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

	result, err := metaengine.ExecuteTyped[projections.FailureLogQuery, []commandlifecycle.FailedPayload](
		context.Background(),
		store,
		projections.FailureLogQuery{Limit: 10},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(HaveLen(3))
	g.Expect(result[0].Error).To(Equal("timeout"))
	g.Expect(result[0].Attempt).To(Equal(1))
	g.Expect(result[2].Attempt).To(Equal(3))
}

func TestProcessingTime_AppliesAndComputesDuration(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		projections.ProcessingTime(),
	)
	g.Expect(err).NotTo(HaveOccurred())

	cmdID := id.NewCommandID()
	received := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	completed := received.Add(150 * time.Millisecond)

	g.Expect(store.ApplyRecord(
		context.Background(),
		makeRecord("command.received", cmdID),
		commandlifecycle.ReceivedPayload{
			CommandID:   commandlifecycle.CommandKey(cmdID.String()),
			CommandType: "create_user",
			ReceivedAt:  received,
		},
	)).To(Succeed())

	g.Expect(store.ApplyRecord(
		context.Background(),
		makeRecord("command.completed", cmdID),
		commandlifecycle.CompletedPayload{
			CommandID:   commandlifecycle.CommandKey(cmdID.String()),
			CommandType: "create_user",
			CompletedAt: completed,
		},
	)).To(Succeed())

	result, err := metaengine.ExecuteTyped[projections.ProcessingTimeQuery, projections.ProcessingTimeEntry](
		context.Background(),
		store,
		projections.ProcessingTimeQuery{CommandID: commandlifecycle.CommandKey(cmdID.String())},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.CommandID).To(Equal(commandlifecycle.CommandKey(cmdID.String())))
	g.Expect(result.ReceivedAt).To(Equal(received))
	g.Expect(result.CompletedAt).To(Equal(completed))
	g.Expect(result.DurationMs).To(Equal(int64(150)))
}

func makeRecord(eventType string, cmdID id.CommandID) record.Record {
	return record.Record{
		Type:       eventType,
		StreamType: string(commandlifecycle.StreamTypeCommandLifecycle),
		StreamID:   record.NewStreamRef("CommandLifecycle", cmdID.String()),
		MetaData: record.CommonMetadata{
			CausationID: cmdID.String(),
			Cause:       record.Cause{Kind: record.CauseCommand, ID: cmdID.String()},
		},
	}
}
