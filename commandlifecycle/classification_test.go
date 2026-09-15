package commandlifecycle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func TestEventTypeConstants_IncludesRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(string(commandlifecycle.TypeRejected)).To(Equal("command.rejected"))
}

func TestDefaultRejectionFamilies(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	families := commandlifecycle.DefaultRejectionFamilies()
	g.Expect(families).To(ConsistOf(errorfamily.Rejection, errorfamily.Conflict))
}

func TestRecorder_IsRejection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		custom []errorfamily.Family
		err    error
		want   bool
	}{
		{name: "nil is never a rejection", err: nil, want: false},
		{
			name: "rejection family",
			err:  errorfamily.NewRejection("USER_EXISTS", "user exists"),
			want: true,
		},
		{
			name: "conflict family",
			err:  errorfamily.NewConflict("VERSION_MISMATCH", "stale version"),
			want: true,
		},
		{
			name: "transient is a failure",
			err:  errorfamily.NewTransient("DB_TIMEOUT", "timeout"),
			want: false,
		},
		{
			name: "corruption is a failure",
			err:  errorfamily.NewCorruption("BAD_PAYLOAD", "bad payload"),
			want: false,
		},
		{name: "unclassified defaults to transient", err: errors.New("boom"), want: false},
		{
			name:   "custom contract widens to transient",
			err:    errorfamily.NewTransient("DB_TIMEOUT", "timeout"),
			want:   true,
			custom: []errorfamily.Family{errorfamily.Transient, errorfamily.Rejection},
		},
		{
			name:   "empty contract disables rejections",
			err:    errorfamily.NewRejection("USER_EXISTS", "user exists"),
			want:   false,
			custom: []errorfamily.Family{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			opts := []commandlifecycle.RecorderOption{}
			if tt.custom != nil {
				opts = append(opts, commandlifecycle.WithRejectionFamilies(tt.custom...))
			}

			recorder := commandlifecycle.NewRecorder(newMemoryStore(t), opts...)
			g.Expect(recorder.IsRejection(tt.err)).To(Equal(tt.want))
		})
	}
}

func TestRecorder_RecordRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	fixed := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	recorder := commandlifecycle.NewRecorder(
		store,
		commandlifecycle.WithClock(func() time.Time { return fixed }),
	)
	cmd := newTestCommand(t)

	err := recorder.RecordRejected(
		context.Background(), cmd,
		errorfamily.NewRejection("USER_EXISTS", "user already exists"),
		1,
	)
	g.Expect(err).NotTo(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(1))
	g.Expect(string(events[0].Type())).To(Equal("command.rejected"))

	payload, err := event.DecodePayloadAuto[commandlifecycle.RejectedPayload](events[0])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(payload.CommandID).To(Equal(commandlifecycle.CommandKey(cmd.ID().String())))
	g.Expect(payload.CommandType).To(Equal("create_user"))
	g.Expect(payload.Error).To(ContainSubstring("user already exists"))
	g.Expect(payload.Family).To(Equal("rejection"))
	g.Expect(payload.ErrorCode).To(Equal("USER_EXISTS"))
	g.Expect(payload.Attempt).To(Equal(1))
	g.Expect(payload.RejectedAt.Equal(fixed)).To(BeTrue())
}

func TestRecorder_RecordFailed_CarriesCommandID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	g.Expect(recorder.RecordFailed(context.Background(), cmd, errors.New("boom"), 1)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(1))

	payload, err := event.DecodePayloadAuto[commandlifecycle.FailedPayload](events[0])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(payload.CommandID).To(Equal(commandlifecycle.CommandKey(cmd.ID().String())))
}
