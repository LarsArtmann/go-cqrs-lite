package event

import (
	"context"
	"testing"
	"time"
)

func TestDeadlineCtx_Deadline(t *testing.T) {
	t.Parallel()

	deadline := time.Now().Add(1 * time.Hour)
	ctx := &deadlineCtx{deadline: deadline}

	got, ok := ctx.Deadline()
	if !ok {
		t.Error("Deadline() ok = false, want true")
	}
	if !got.Equal(deadline) {
		t.Errorf("Deadline() = %v, want %v", got, deadline)
	}
}

func TestDeadlineCtx_Done_NotExpired(t *testing.T) {
	t.Parallel()

	ctx := &deadlineCtx{deadline: time.Now().Add(1 * time.Hour)}

	ch := ctx.Done()
	if ch != nil {
		t.Error("Done() should return nil when not expired")
	}
}

func TestDeadlineCtx_Done_Expired(t *testing.T) {
	t.Parallel()

	ctx := &deadlineCtx{deadline: time.Now().Add(-1 * time.Second)}

	ch := ctx.Done()
	if ch == nil {
		t.Fatal("Done() should return non-nil channel when expired")
	}

	select {
	case <-ch:
	default:
		t.Error("Done() channel should be closed when expired")
	}
}

func TestDeadlineCtx_Err_NotExpired(t *testing.T) {
	t.Parallel()

	ctx := &deadlineCtx{deadline: time.Now().Add(1 * time.Hour)}

	if err := ctx.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestDeadlineCtx_Err_Expired(t *testing.T) {
	t.Parallel()

	ctx := &deadlineCtx{deadline: time.Now().Add(-1 * time.Second)}

	if err := ctx.Err(); err != context.DeadlineExceeded {
		t.Errorf("Err() = %v, want DeadlineExceeded", err)
	}
}

func TestDeadlineCtx_Value(t *testing.T) {
	t.Parallel()

	ctx := &deadlineCtx{deadline: time.Now()}

	if v := ctx.Value("any"); v != nil {
		t.Errorf("Value() = %v, want nil", v)
	}
}
