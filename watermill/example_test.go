package watermill_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	wmlog "github.com/ThreeDotsLabs/watermill"
	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func ExampleNewSubscriberAdapter() {
	bus := eventtest.NewFakeBus()
	adapter := watermill.NewSubscriberAdapter(bus)

	fmt.Println(adapter != nil)

	// Output:
	// true
}

func ExampleNewPublisherAdapter() {
	bus := eventtest.NewFakeBus()
	adapter := watermill.NewPublisherAdapter(bus)

	fmt.Println(adapter != nil)

	// Output:
	// true
}

func ExampleNewCommandBus() {
	bus := watermill.NewCommandBus()
	defer bus.Close()

	var received atomic.Int32

	_ = bus.Subscribe("user.create", func(_ context.Context, _ command.Command) error {
		received.Add(1)

		return nil
	})

	cmd, _ := command.New("user.create", id.NewAggregateID())
	_ = bus.Publish(context.Background(), cmd)

	time.Sleep(50 * time.Millisecond)
	fmt.Println(received.Load())

	// Output:
	// 1
}

func ExampleNewCommandPublisher() {
	// CommandPublisher wraps any Watermill message.Publisher as a command.Publisher
	wmPub := gochannel.NewGoChannel(
		gochannel.Config{BlockPublishUntilSubscriberAck: true},
		wmlog.NopLogger{},
	)
	defer wmPub.Close() //nolint:errcheck // example cleanup

	pub := watermill.NewCommandPublisher(wmPub, "commands")
	fmt.Println(pub != nil)

	// Output:
	// true
}

func ExampleCommandToMessage() {
	cmd, _ := command.New("user.create", id.NewAggregateID())
	msg := watermill.CommandToMessage(cmd)

	fmt.Println(msg.Metadata.Get("command_type"))

	// Output:
	// user.create
}
