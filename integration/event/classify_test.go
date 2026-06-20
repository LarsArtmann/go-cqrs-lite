package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

func TestClassify_DeciderSentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(decider.ErrNilApply) != event.Rejection {
		t.Error("decider.ErrNilApply should be Rejection")
	}

	if event.Classify(decider.ErrNilStore) != event.Infrastructure {
		t.Error("decider.ErrNilStore should be Infrastructure")
	}

	if event.Classify(decider.ErrNilBus) != event.Infrastructure {
		t.Error("decider.ErrNilBus should be Infrastructure")
	}
}

func TestClassify_StorageSentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(storage.ErrNilDB) != event.Infrastructure {
		t.Error("storage.ErrNilDB should be Infrastructure")
	}
}

func TestClassify_MiddlewareSentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(middleware.ErrValidationFailed) != event.Rejection {
		t.Error("middleware.ErrValidationFailed should be Rejection")
	}

	if event.Classify(middleware.ErrRetryExhausted) != event.Infrastructure {
		t.Error("middleware.ErrRetryExhausted should be Infrastructure")
	}

	if event.Classify(middleware.ErrRetryCanceled) != event.Infrastructure {
		t.Error("middleware.ErrRetryCanceled should be Infrastructure")
	}

	if event.Classify(middleware.ErrPanicRecovered) != event.Corruption {
		t.Error("middleware.ErrPanicRecovered should be Corruption")
	}
}

func TestClassify_CommandQuerySentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(command.ErrHandlerNotFound) != event.Rejection {
		t.Error("command.ErrHandlerNotFound should be Rejection")
	}

	if event.Classify(command.ErrDispatcherClosed) != event.Infrastructure {
		t.Error("command.ErrDispatcherClosed should be Infrastructure")
	}

	if event.Classify(query.ErrHandlerNotFound) != event.Rejection {
		t.Error("query.ErrHandlerNotFound should be Rejection")
	}

	if event.Classify(query.ErrDispatcherClosed) != event.Infrastructure {
		t.Error("query.ErrDispatcherClosed should be Infrastructure")
	}
}
