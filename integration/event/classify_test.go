package event_test

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func TestClassify_DeciderSentinels(t *testing.T) {
	t.Parallel()

	if errorfamily.Classify(decider.ErrNilApply) != errorfamily.Rejection {
		t.Error("decider.ErrNilApply should be Rejection")
	}

	if errorfamily.Classify(decider.ErrNilStore) != errorfamily.Infrastructure {
		t.Error("decider.ErrNilStore should be Infrastructure")
	}

	if errorfamily.Classify(decider.ErrNilBus) != errorfamily.Infrastructure {
		t.Error("decider.ErrNilBus should be Infrastructure")
	}
}

func TestClassify_StorageSentinels(t *testing.T) {
	t.Parallel()

	if errorfamily.Classify(storage.ErrNilDB) != errorfamily.Infrastructure {
		t.Error("storage.ErrNilDB should be Infrastructure")
	}
}

func TestClassify_MiddlewareSentinels(t *testing.T) {
	t.Parallel()

	if errorfamily.Classify(middleware.ErrValidationFailed) != errorfamily.Rejection {
		t.Error("middleware.ErrValidationFailed should be Rejection")
	}

	if errorfamily.Classify(middleware.ErrRetryExhausted) != errorfamily.Infrastructure {
		t.Error("middleware.ErrRetryExhausted should be Infrastructure")
	}

	if errorfamily.Classify(middleware.ErrRetryCanceled) != errorfamily.Infrastructure {
		t.Error("middleware.ErrRetryCanceled should be Infrastructure")
	}

	if errorfamily.Classify(middleware.ErrPanicRecovered) != errorfamily.Corruption {
		t.Error("middleware.ErrPanicRecovered should be Corruption")
	}
}

func TestClassify_CommandQuerySentinels(t *testing.T) {
	t.Parallel()

	if errorfamily.Classify(command.ErrHandlerNotFound) != errorfamily.Rejection {
		t.Error("command.ErrHandlerNotFound should be Rejection")
	}

	if errorfamily.Classify(command.ErrDispatcherClosed) != errorfamily.Infrastructure {
		t.Error("command.ErrDispatcherClosed should be Infrastructure")
	}

	if errorfamily.Classify(query.ErrHandlerNotFound) != errorfamily.Rejection {
		t.Error("query.ErrHandlerNotFound should be Rejection")
	}

	if errorfamily.Classify(query.ErrDispatcherClosed) != errorfamily.Infrastructure {
		t.Error("query.ErrDispatcherClosed should be Infrastructure")
	}
}
