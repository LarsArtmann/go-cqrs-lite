package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/middleware"
	"github.com/larsartmann/go-cqrs-lite/projection"
	"github.com/larsartmann/go-cqrs-lite/storage"
)

func TestClassify_AggregateSentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(aggregate.ErrNilAggregateID) != event.Rejection {
		t.Error("aggregate.ErrNilAggregateID should be Rejection")
	}

	if event.Classify(aggregate.ErrEmptyAggregateType) != event.Rejection {
		t.Error("aggregate.ErrEmptyAggregateType should be Rejection")
	}

	if event.Classify(aggregate.ErrNilStore) != event.Infrastructure {
		t.Error("aggregate.ErrNilStore should be Infrastructure")
	}

	if event.Classify(aggregate.ErrNilBus) != event.Infrastructure {
		t.Error("aggregate.ErrNilBus should be Infrastructure")
	}
}

func TestClassify_ProjectionSentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(projection.ErrNilHandler) != event.Rejection {
		t.Error("projection.ErrNilHandler should be Rejection")
	}

	if event.Classify(projection.ErrNilBus) != event.Infrastructure {
		t.Error("projection.ErrNilBus should be Infrastructure")
	}

	if event.Classify(projection.ErrNilCheckpoint) != event.Infrastructure {
		t.Error("projection.ErrNilCheckpoint should be Infrastructure")
	}

	if event.Classify(projection.ErrNoProjections) != event.Rejection {
		t.Error("projection.ErrNoProjections should be Rejection")
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
